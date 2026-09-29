// Package proxyintake accepts forward-proxy requests for web pages and hands
// them to the relay.
package proxyintake

import (
	"context"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type Relay interface {
	ReplyTo(
		ctx context.Context,
		method string,
		address canonicalurl.CanonicalURL,
		requestHeaders http.Header,
		reply requestrelay.Reply,
	)
}

type Observer interface {
	MethodRefused(ctx context.Context, method string)
	TargetRefused(ctx context.Context, target string, cause error)
}

type Endpoint struct {
	relay    Relay
	observer Observer
}

func New(relay Relay, observer Observer) *Endpoint {
	return &Endpoint{relay: relay, observer: observer}
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
	e.relay.ReplyTo(
		r.Context(),
		r.Method,
		address,
		r.Header,
		clientReply{Writer: w, responseWriter: w},
	)
}

type clientReply struct {
	io.Writer
	responseWriter http.ResponseWriter
}

func (c clientReply) SendHead(status int, headers http.Header) {
	replyHeaders := c.responseWriter.Header()
	for name, values := range headers {
		replyHeaders[name] = values
	}
	for _, name := range []string{"Content-Type", "Date"} {
		if _, found := replyHeaders[name]; !found {
			replyHeaders[name] = nil
		}
	}
	c.responseWriter.WriteHeader(status)
}

func (c clientReply) CutShort() {
	_ = http.NewResponseController(c.responseWriter).Flush()
	panic(http.ErrAbortHandler)
}
