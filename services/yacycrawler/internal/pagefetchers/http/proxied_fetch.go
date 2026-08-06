package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

const (
	headerUserAgent       = "User-Agent"
	headerContentType     = "Content-Type"
	headerRetryAfter      = "Retry-After"
	headerXRobotsTag      = "X-Robots-Tag"
	headerETag            = "ETag"
	headerLastModified    = "Last-Modified"
	headerIfNoneMatch     = "If-None-Match"
	headerIfModifiedSince = "If-Modified-Since"

	defaultDeferFor = time.Minute

	msgFetchTransient = "fetch failed, treating as transient"
	msgBodyReadFailed = "response body read failed, treating as transient"
)

type ProxiedFetch struct {
	client       *http.Client
	userAgent    string
	maxBodyBytes int64
	deadline     time.Duration
}

func New(
	proxyURL *url.URL,
	dialMode ProxyDialMode,
	userAgent string,
	maxBodyBytes int64,
	deadline time.Duration,
) *ProxiedFetch {
	return &ProxiedFetch{
		client:       &http.Client{Transport: transportForDialMode(proxyURL, dialMode)},
		userAgent:    userAgent,
		maxBodyBytes: maxBodyBytes,
		deadline:     deadline,
	}
}

func (f *ProxiedFetch) Fetch(
	ctx context.Context,
	rawURL string,
	knownVersion pagevisit.PageVersion,
) (pagevisit.FetchOutcome, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, f.deadline)
	defer cancel()

	request, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return pagevisit.FetchOutcome{}, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set(headerUserAgent, f.userAgent)
	setConditionalHeaders(request, knownVersion)

	response, err := f.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return pagevisit.FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, ctx.Err())
		}
		slog.WarnContext(ctx, msgFetchTransient,
			slog.String("url", rawURL),
			slog.Any("error", err),
		)
		return pagevisit.FetchOutcome{Status: pagevisit.FetchFailed}, nil
	}
	defer func() { _ = response.Body.Close() }()

	return f.classify(ctx, response, knownVersion)
}

func (f *ProxiedFetch) classify(
	ctx context.Context,
	response *http.Response,
	sent pagevisit.PageVersion,
) (pagevisit.FetchOutcome, error) {
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return f.fetched(ctx, response)
	case response.StatusCode == http.StatusNotModified:
		return pagevisit.FetchOutcome{
			Status:  pagevisit.FetchNotModified,
			Version: sent,
		}, nil
	case response.StatusCode == http.StatusTooManyRequests,
		response.StatusCode == http.StatusServiceUnavailable:
		return pagevisit.FetchOutcome{
			Status:   pagevisit.FetchDeferred,
			DeferFor: retryAfter(response.Header.Get(headerRetryAfter)),
		}, nil
	case response.StatusCode == http.StatusUnauthorized,
		response.StatusCode == http.StatusForbidden,
		response.StatusCode == http.StatusUnavailableForLegalReasons:
		return pagevisit.FetchOutcome{Status: pagevisit.FetchCeased}, nil
	case response.StatusCode >= 400 && response.StatusCode < 500:
		return pagevisit.FetchOutcome{Status: pagevisit.FetchNotAPage}, nil
	default:
		return pagevisit.FetchOutcome{Status: pagevisit.FetchFailed}, nil
	}
}

func (f *ProxiedFetch) fetched(
	ctx context.Context,
	response *http.Response,
) (pagevisit.FetchOutcome, error) {
	body, readErr := readBody(response.Body, f.maxBodyBytes+1)
	if readErr != nil {
		slog.WarnContext(ctx, msgBodyReadFailed,
			slog.String("url", response.Request.URL.String()),
			slog.Any("error", readErr),
		)
		return pagevisit.FetchOutcome{Status: pagevisit.FetchFailed}, nil
	}
	truncated := int64(len(body)) > f.maxBodyBytes
	if truncated {
		body = body[:f.maxBodyBytes]
	}
	noIndex, noFollow := robotsDirectives(response.Header.Values(headerXRobotsTag))
	return pagevisit.FetchOutcome{
		Status: pagevisit.FetchSucceeded,
		Page: fetchedpage.Page{
			FinalURL:             response.Request.URL.String(),
			ContentType:          response.Header.Get(headerContentType),
			Body:                 body,
			Truncated:            truncated,
			RefusesIndexing:      noIndex,
			RefusesLinkDiscovery: noFollow,
		},
		RedirectChain: redirectChain(response),
		Version:       pageVersionOf(response),
	}, nil
}

func setConditionalHeaders(request *http.Request, version pagevisit.PageVersion) {
	if version.EntityTag != "" {
		request.Header.Set(headerIfNoneMatch, version.EntityTag)
	}
	if !version.ModifiedAt.IsZero() {
		request.Header.Set(
			headerIfModifiedSince,
			version.ModifiedAt.UTC().Format(http.TimeFormat),
		)
	}
}

func pageVersionOf(response *http.Response) pagevisit.PageVersion {
	version := pagevisit.PageVersion{EntityTag: response.Header.Get(headerETag)}
	if modified, err := http.ParseTime(response.Header.Get(headerLastModified)); err == nil {
		version.ModifiedAt = modified
	}
	return version
}

func redirectChain(response *http.Response) []string {
	var chain []string
	for request := response.Request; request != nil; {
		chain = append(chain, request.URL.String())
		if request.Response == nil {
			break
		}
		request = request.Response.Request
	}
	for left, right := 0, len(chain)-1; left < right; left, right = left+1, right-1 {
		chain[left], chain[right] = chain[right], chain[left]
	}
	return chain
}

func readBody(source io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(source, limit))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func robotsDirectives(values []string) (noIndex, noFollow bool) {
	for _, value := range values {
		for _, directive := range strings.Split(value, ",") {
			switch strings.ToLower(strings.TrimSpace(directive)) {
			case "noindex":
				noIndex = true
			case "nofollow":
				noFollow = true
			case "none":
				noIndex = true
				noFollow = true
			}
		}
	}
	return noIndex, noFollow
}

func retryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return defaultDeferFor
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds < 0 {
			return defaultDeferFor
		}
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		if wait := time.Until(when); wait > 0 {
			return wait
		}
		return 0
	}
	return defaultDeferFor
}
