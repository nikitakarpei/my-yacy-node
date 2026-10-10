// Package proxyintake accepts forward-proxy requests for web pages, and
// answers each with the page that Chrome would get.
package proxyintake

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
)

var hopByHopHeaderNames = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

type PageFetcher interface {
	Fetch(ctx context.Context, request chromenavigation.Request) (*http.Response, bool)
}

type Endpoint struct {
	navigator   *chromenavigation.Navigator
	pageFetcher PageFetcher
	observers   Observers
}

func New(
	navigator *chromenavigation.Navigator,
	pageFetcher PageFetcher,
	observers Observers,
) *Endpoint {
	return &Endpoint{navigator: navigator, pageFetcher: pageFetcher, observers: observers}
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		e.observers.MethodRefused(r.Context(), r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isWebAddress(r.URL) {
		e.observers.TargetRefused(r.Context(), r.RequestURI)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	page, wasFetched := e.pageFetcher.Fetch(
		r.Context(),
		e.navigator.RequestFor(r.Method, r.URL, r.Header),
	)
	if !wasFetched {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer func() { _ = page.Body.Close() }()
	e.relay(r.Context(), r.URL.String(), page, w)
}

func isWebAddress(address *url.URL) bool {
	return (address.Scheme == "http" || address.Scheme == "https") && address.Host != ""
}

func (e *Endpoint) relay(
	ctx context.Context,
	address string,
	page *http.Response,
	w http.ResponseWriter,
) {
	sendHeaders(w, page.StatusCode, endToEndHeadersOf(page.Header))
	clientWrites := &clientWriter{writer: w}
	if _, err := io.Copy(clientWrites, page.Body); err != nil {
		e.observers.ResponseLeftIncomplete(
			ctx,
			address,
			incompleteResponseCauseOf(ctx, clientWrites.writeFailed),
			err,
		)
		_ = http.NewResponseController(w).Flush()
		panic(http.ErrAbortHandler)
	}
}

func sendHeaders(w http.ResponseWriter, status int, headers http.Header) {
	responseHeaders := w.Header()
	for name, values := range headers {
		responseHeaders[name] = values
	}
	for _, name := range []string{"Content-Type", "Date"} {
		if _, found := responseHeaders[name]; !found {
			responseHeaders[name] = nil
		}
	}
	w.WriteHeader(status)
}

func endToEndHeadersOf(headers http.Header) http.Header {
	endToEndHeaders := headers.Clone()
	for _, value := range headers.Values("Connection") {
		for name := range strings.SplitSeq(value, ",") {
			endToEndHeaders.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range hopByHopHeaderNames {
		endToEndHeaders.Del(name)
	}
	return endToEndHeaders
}

func incompleteResponseCauseOf(
	ctx context.Context,
	clientWriteFailed bool,
) IncompleteResponseCause {
	if clientWriteFailed || ctx.Err() != nil {
		return ClientClosedRequest
	}
	return PageReadFailed
}
