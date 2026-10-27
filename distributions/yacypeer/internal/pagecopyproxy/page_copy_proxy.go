// Package pagecopyproxy runs an HTTP forward proxy for page fetches on the loopback interface.
// It answers a page fetch from a page copy when one exists. Else it fetches the page through the
// proxy it is given, relays the answer, and keeps a copy of it.
package pagecopyproxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pagecopies"
)

const (
	Name                  = "pagecopyproxy"
	shutdownLimit         = 10 * time.Second
	readHeaderLimit       = 10 * time.Second
	originFetchLimit      = 30 * time.Second
	msgFetchRefused       = "page copy proxy takes GET with an absolute URL"
	msgOriginFetchFailed  = "page fetch failed at the origin"
	headerUserAgent       = "User-Agent"
	headerIfNoneMatch     = "If-None-Match"
	headerIfModifiedSince = "If-Modified-Since"
	headerLocation        = "Location"
	headerContentLength   = "Content-Length"
)

var unrelayedHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Proxy-Connection",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade", "Set-Cookie", headerContentLength,
}

type PageCopyProxy struct {
	listener     net.Listener
	copies       *pagecopies.Copies
	originClient *http.Client
	maxBodyBytes int64
}

func Listen(
	ctx context.Context,
	copies *pagecopies.Copies,
	originProxyURL *url.URL,
	maxBodyBytes int64,
) (*PageCopyProxy, error) {
	listener, err := loopback.Listen(ctx)
	if err != nil {
		return nil, fmt.Errorf("page copy proxy: %w", err)
	}
	return &PageCopyProxy{
		listener: listener,
		copies:   copies,
		originClient: &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(originProxyURL)},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		maxBodyBytes: maxBodyBytes,
	}, nil
}

func (p *PageCopyProxy) URL() string {
	return loopback.URLOf(p.listener)
}

func (p *PageCopyProxy) Close() error {
	if err := p.listener.Close(); err != nil {
		return fmt.Errorf("page copy proxy: %w", err)
	}
	return nil
}

func (p *PageCopyProxy) Run(ctx context.Context) error {
	return servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{{
		Name:     Name,
		Listener: p.listener,
		Server: &http.Server{
			Handler:           p,
			ReadHeaderTimeout: readHeaderLimit,
		},
	}})
}

func (p *PageCopyProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fetch, isFetchable := pageFetchFrom(r)
	if !isFetchable {
		http.Error(w, msgFetchRefused, http.StatusMethodNotAllowed)
		return
	}
	if p.answerFromCopy(r.Context(), w, fetch) {
		return
	}
	p.answerFromOrigin(r.Context(), w, fetch)
}

type pageFetch struct {
	pageURL         string
	userAgent       string
	ifNoneMatch     string
	ifModifiedSince string
}

func pageFetchFrom(r *http.Request) (pageFetch, bool) {
	fetch := pageFetch{
		pageURL:         r.URL.String(),
		userAgent:       r.Header.Get(headerUserAgent),
		ifNoneMatch:     r.Header.Get(headerIfNoneMatch),
		ifModifiedSince: r.Header.Get(headerIfModifiedSince),
	}
	return fetch, r.Method == http.MethodGet && r.URL.IsAbs()
}

func (f pageFetch) copyKey() string {
	return f.userAgent + "\n" + f.pageURL
}

func (f pageFetch) isConditional() bool {
	return f.ifNoneMatch != "" || f.ifModifiedSince != ""
}

func (p *PageCopyProxy) answerFromCopy(
	ctx context.Context,
	w http.ResponseWriter,
	fetch pageFetch,
) (wasAnswered bool) {
	if fetch.isConditional() {
		return false
	}
	pageCopy, found := p.copies.CopyFor(ctx, fetch.copyKey())
	if !found {
		return false
	}
	writeWholeAnswer(w, pageCopy)
	logAnswer(ctx, fetch, "copy", pageCopy.StatusCode, int64(len(pageCopy.Body)))
	return true
}

func writeWholeAnswer(w http.ResponseWriter, pageCopy pagecopies.PageCopy) {
	writeHeaderOf(w, pageCopy)
	w.Header().Set(headerContentLength, strconv.Itoa(len(pageCopy.Body)))
	w.WriteHeader(pageCopy.StatusCode)
	_, _ = w.Write(pageCopy.Body)
}

func writeHeaderOf(w http.ResponseWriter, pageCopy pagecopies.PageCopy) {
	for name, values := range pageCopy.Header {
		w.Header()[name] = values
	}
}

func logAnswer(
	ctx context.Context,
	fetch pageFetch,
	source string,
	statusCode int,
	bodyBytes int64,
) {
	slog.DebugContext(ctx, "page fetch answered",
		slog.String("pageUrl", fetch.pageURL),
		slog.String("source", source),
		slog.Int("statusCode", statusCode),
		slog.Int64("bodyBytes", bodyBytes),
		slog.Bool("conditional", fetch.isConditional()),
	)
}

func (p *PageCopyProxy) answerFromOrigin(
	ctx context.Context,
	w http.ResponseWriter,
	fetch pageFetch,
) {
	answer, err := p.originAnswerTo(ctx, fetch)
	if err != nil {
		reportOriginFailure(ctx, w, fetch, err)
		return
	}
	defer answer.release()
	if !fetch.isConditional() && p.isKeepable(answer) {
		p.copies.Keep(ctx, fetch.copyKey(), answer.pageCopy)
	}
	bodyBytes := p.writeOriginAnswer(w, answer)
	logAnswer(ctx, fetch, "origin", answer.pageCopy.StatusCode, bodyBytes)
}

type originAnswer struct {
	pageCopy      pagecopies.PageCopy
	remainingBody io.Reader
	release       func()
}

func (p *PageCopyProxy) originAnswerTo(ctx context.Context, fetch pageFetch) (originAnswer, error) {
	ctx, cancel := context.WithTimeout(ctx, originFetchLimit)
	request, err := originRequestFor(ctx, fetch)
	if err != nil {
		cancel()
		return originAnswer{}, err
	}
	//nolint:gosec // G704: the origin proxy makes the connection.
	response, err := p.originClient.Do(request)
	if err != nil {
		cancel()
		return originAnswer{}, fmt.Errorf("fetch the origin: %w", err)
	}
	release := func() {
		_ = response.Body.Close()
		cancel()
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, p.maxBodyBytes+1))
	if err != nil {
		release()
		return originAnswer{}, fmt.Errorf("read the origin answer: %w", err)
	}
	header := response.Header.Clone()
	for _, name := range unrelayedHeaders {
		header.Del(name)
	}
	return originAnswer{
		pageCopy: pagecopies.PageCopy{
			Header:     header,
			Body:       body,
			StatusCode: response.StatusCode,
		},
		remainingBody: response.Body,
		release:       release,
	}, nil
}

func originRequestFor(ctx context.Context, fetch pageFetch) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fetch.pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build the origin request: %w", err)
	}
	request.Header.Set(headerUserAgent, fetch.userAgent)
	if fetch.ifNoneMatch != "" {
		request.Header.Set(headerIfNoneMatch, fetch.ifNoneMatch)
	}
	if fetch.ifModifiedSince != "" {
		request.Header.Set(headerIfModifiedSince, fetch.ifModifiedSince)
	}
	return request, nil
}

func reportOriginFailure(ctx context.Context, w http.ResponseWriter, fetch pageFetch, err error) {
	if ctx.Err() != nil {
		slog.DebugContext(ctx, "page fetch abandoned", slog.String("pageUrl", fetch.pageURL))
		return
	}
	slog.WarnContext(ctx, msgOriginFetchFailed,
		slog.String("pageUrl", fetch.pageURL), slog.Any("error", err))
	http.Error(w, msgOriginFetchFailed, http.StatusBadGateway)
}

func (p *PageCopyProxy) isKeepable(answer originAnswer) bool {
	if p.isCutShort(answer) {
		return false
	}
	switch statusCode := answer.pageCopy.StatusCode; {
	case statusCode >= 200 && statusCode < 300:
		return true
	case statusCode >= 300 && statusCode < 400:
		return answer.pageCopy.Header.Get(headerLocation) != ""
	default:
		return statusCode == http.StatusNotFound || statusCode == http.StatusGone
	}
}

func (p *PageCopyProxy) isCutShort(answer originAnswer) bool {
	return int64(len(answer.pageCopy.Body)) > p.maxBodyBytes
}

func (p *PageCopyProxy) writeOriginAnswer(w http.ResponseWriter, answer originAnswer) int64 {
	if !p.isCutShort(answer) {
		writeWholeAnswer(w, answer.pageCopy)
		return int64(len(answer.pageCopy.Body))
	}
	writeHeaderOf(w, answer.pageCopy)
	w.WriteHeader(answer.pageCopy.StatusCode)
	_, _ = w.Write(answer.pageCopy.Body)
	remainingBytes, _ := io.Copy(w, answer.remainingBody)
	return int64(len(answer.pageCopy.Body)) + remainingBytes
}
