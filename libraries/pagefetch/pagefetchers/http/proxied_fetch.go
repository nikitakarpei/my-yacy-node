package http

import (
	"context"
	"fmt"
	"io"
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
	headerLocation        = "Location"

	defaultDeferFor = time.Minute
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
		client: &http.Client{
			Transport:     transportForDialMode(proxyURL, dialMode),
			CheckRedirect: relayRedirect,
		},
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
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed, FailureCause: err}, nil
	}
	defer func() { _ = response.Body.Close() }()

	return f.classify(response, knownVersion), nil
}

func (f *ProxiedFetch) classify(
	response *http.Response,
	sent pagefetch.PageVersion,
) pagefetch.FetchOutcome {
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		return f.fetched(response)
	case response.StatusCode == http.StatusNotModified:
		return pagefetch.FetchOutcome{
			Status:  pagefetch.FetchNotModified,
			Version: sent,
		}
	case response.StatusCode >= 300 && response.StatusCode < 400 &&
		response.Header.Get(headerLocation) != "":
		return redirected(response)
	case response.StatusCode == http.StatusTooManyRequests,
		response.StatusCode == http.StatusServiceUnavailable:
		return pagefetch.FetchOutcome{
			Status:   pagefetch.FetchDeferred,
			DeferFor: retryAfter(response.Header.Get(headerRetryAfter)),
		}
	case response.StatusCode == http.StatusUnauthorized,
		response.StatusCode == http.StatusForbidden,
		response.StatusCode == http.StatusUnavailableForLegalReasons:
		return pagefetch.FetchOutcome{Status: pagefetch.FetchAccessRefused}
	case response.StatusCode >= 400 && response.StatusCode < 500:
		return pagefetch.FetchOutcome{Status: pagefetch.FetchRejected}
	default:
		return pagefetch.FetchOutcome{
			Status:       pagefetch.FetchFailed,
			FailureCause: fmt.Errorf("origin response: %s", response.Status),
		}
	}
}

func (f *ProxiedFetch) fetched(
	response *http.Response,
) pagefetch.FetchOutcome {
	body, readErr := readBody(response.Body, f.maxBodyBytes+1)
	if readErr != nil {
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed, FailureCause: readErr}
	}
	if int64(len(body)) > f.maxBodyBytes {
		return pagefetch.FetchOutcome{Status: pagefetch.FetchOversized}
	}
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page: pagefetch.FetchedPage{
			ContentType:      response.Header.Get(headerContentType),
			Body:             body,
			RobotsDirectives: response.Header.Values(headerXRobotsTag),
		},
		Version: pageVersionOf(response),
	}
}

func redirected(response *http.Response) pagefetch.FetchOutcome {
	location, err := response.Request.URL.Parse(response.Header.Get(headerLocation))
	if err != nil {
		return pagefetch.FetchOutcome{
			Status:       pagefetch.FetchRedirectTargetInvalid,
			FailureCause: fmt.Errorf("resolve location: %w", err),
		}
	}
	redirectTarget, err := canonicalurl.CanonicalURLOf(location.String())
	if err != nil {
		return pagefetch.FetchOutcome{
			Status:       pagefetch.FetchRedirectTargetInvalid,
			FailureCause: err,
		}
	}
	return pagefetch.FetchOutcome{
		Status:         pagefetch.FetchRedirected,
		RedirectTarget: redirectTarget,
		Version:        pageVersionOf(response),
	}
}

func relayRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
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
	version := pagefetch.PageVersion{EntityTag: response.Header.Get(headerETag)}
	if modified, err := http.ParseTime(response.Header.Get(headerLastModified)); err == nil {
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
