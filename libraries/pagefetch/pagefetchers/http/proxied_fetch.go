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

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
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

	msgFetchTransient   = "fetch failed, treating as transient"
	msgBodyReadFailed   = "response body read failed, treating as transient"
	msgFinalURLRejected = "fetched page url rejected, page treated as no page"
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
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, f.deadline)
	defer cancel()

	request, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, pageURL.String(), nil)
	if err != nil {
		return pagefetch.FetchOutcome{}, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set(headerUserAgent, f.userAgent)
	setConditionalHeaders(request, knownVersion)

	response, err := f.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return pagefetch.FetchOutcome{}, fmt.Errorf("fetch %s: %w", pageURL, ctx.Err())
		}
		slog.WarnContext(ctx, msgFetchTransient,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}, nil
	}
	defer func() { _ = response.Body.Close() }()

	return f.classify(ctx, response, knownVersion)
}

func (f *ProxiedFetch) classify(
	ctx context.Context,
	response *http.Response,
	sent pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return f.fetched(ctx, response)
	case response.StatusCode == http.StatusNotModified:
		return pagefetch.FetchOutcome{
			Status:  pagefetch.FetchNotModified,
			Version: sent,
		}, nil
	case response.StatusCode == http.StatusTooManyRequests,
		response.StatusCode == http.StatusServiceUnavailable:
		return pagefetch.FetchOutcome{
			Status:   pagefetch.FetchDeferred,
			DeferFor: retryAfter(response.Header.Get(headerRetryAfter)),
		}, nil
	case response.StatusCode == http.StatusUnauthorized,
		response.StatusCode == http.StatusForbidden,
		response.StatusCode == http.StatusUnavailableForLegalReasons:
		return pagefetch.FetchOutcome{Status: pagefetch.FetchCeased}, nil
	case response.StatusCode >= 400 && response.StatusCode < 500:
		return pagefetch.FetchOutcome{Status: pagefetch.FetchNotAPage}, nil
	default:
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}, nil
	}
}

func (f *ProxiedFetch) fetched(
	ctx context.Context,
	response *http.Response,
) (pagefetch.FetchOutcome, error) {
	body, readErr := readBody(response.Body, f.maxBodyBytes+1)
	if readErr != nil {
		slog.WarnContext(ctx, msgBodyReadFailed,
			slog.String("url", response.Request.URL.String()),
			slog.Any("error", readErr),
		)
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}, nil
	}
	truncated := int64(len(body)) > f.maxBodyBytes
	if truncated {
		body = body[:f.maxBodyBytes]
	}
	if capture, replayed := replayedCaptureOf(ctx, response); replayed {
		return succeededFetchOf(
			response, capture.OriginalURL, capture.OriginVersion, body, truncated,
		), nil
	}
	finalURL, err := canonicalurl.CanonicalURLOf(response.Request.URL.String())
	if err != nil {
		slog.WarnContext(ctx, msgFinalURLRejected,
			slog.String("url", response.Request.URL.String()),
			slog.Any("error", err),
		)
		return pagefetch.FetchOutcome{Status: pagefetch.FetchNotAPage}, nil
	}
	return succeededFetchOf(
		response, finalURL, pageVersionOf(response), body, truncated,
	), nil
}

func succeededFetchOf(
	response *http.Response,
	finalURL canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
	body []byte,
	truncated bool,
) pagefetch.FetchOutcome {
	noIndex, noFollow := robotsDirectives(response.Header.Values(headerXRobotsTag))
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			FinalURL:             finalURL,
			ContentType:          response.Header.Get(headerContentType),
			Body:                 body,
			Truncated:            truncated,
			RefusesIndexing:      noIndex,
			RefusesLinkDiscovery: noFollow,
		},
		Version: version,
	}
}

func setConditionalHeaders(request *http.Request, version pagefetch.PageVersion) {
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

func pageVersionOf(response *http.Response) pagefetch.PageVersion {
	return pageVersionFrom(
		response.Header.Get(headerETag),
		response.Header.Get(headerLastModified),
	)
}

func pageVersionFrom(entityTag, lastModified string) pagefetch.PageVersion {
	version := pagefetch.PageVersion{EntityTag: entityTag}
	if modified, err := http.ParseTime(lastModified); err == nil {
		version.ModifiedAt = modified
	}
	return version
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
