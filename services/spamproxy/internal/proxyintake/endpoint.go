// Package proxyintake accepts forward-proxy requests for web pages and hands
// them to the relayer.
package proxyintake

import (
	"context"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type Relayer interface {
	RespondTo(
		ctx context.Context,
		method string,
		address canonicalurl.CanonicalURL,
		requestHeaders http.Header,
		responseWriter requestrelay.ResponseWriter,
	)
}

type Observer interface {
	MethodRefused(ctx context.Context, method string)
	TargetRefused(ctx context.Context, target string, cause error)
}

type Endpoint struct {
	relayer  Relayer
	observer Observer
}

func New(relayer Relayer, observer Observer) *Endpoint {
	return &Endpoint{relayer: relayer, observer: observer}
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		e.observer.MethodRefused(r.Context(), r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	address, err := canonicalurl.CanonicalURLOf(r.RequestURI)
	if err != nil {
		e.observer.TargetRefused(r.Context(), r.RequestURI, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	e.relayer.RespondTo(
		r.Context(),
		r.Method,
		address,
		r.Header,
		clientResponseWriter{Writer: w, httpResponseWriter: w},
	)
}

type clientResponseWriter struct {
	io.Writer
	httpResponseWriter http.ResponseWriter
}

func (c clientResponseWriter) SendHeaders(status int, headers http.Header) {
	responseHeaders := c.httpResponseWriter.Header()
	for name, values := range headers {
		responseHeaders[name] = values
	}
	for _, name := range []string{"Content-Type", "Date"} {
		if _, found := responseHeaders[name]; !found {
			responseHeaders[name] = nil
		}
	}
	c.httpResponseWriter.WriteHeader(status)
}

func (c clientResponseWriter) Abort() {
	_ = http.NewResponseController(c.httpResponseWriter).Flush()
	panic(http.ErrAbortHandler)
}
