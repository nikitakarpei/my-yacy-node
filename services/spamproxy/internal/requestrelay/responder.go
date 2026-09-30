// Package requestrelay relays each proxied request upstream, and sends the
// upstream response to the client.
package requestrelay

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/responsedeadline"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
)

type Upstream interface {
	RoundTrip(ctx context.Context, request proxiedrequest.Request) (*http.Response, error)
}

type Clock interface {
	Now() time.Time
}

type ResponseWriter interface {
	io.Writer
	SendHeaders(status int, headers http.Header)
	Abort()
}

type Responder struct {
	upstream              Upstream
	observers             Observers
	responseHeaderTimeout time.Duration
	clock                 Clock
}

func New(
	upstream Upstream,
	responseHeaderTimeout time.Duration,
	clock Clock,
	observers Observers,
) *Responder {
	return &Responder{
		upstream:              upstream,
		observers:             observers,
		responseHeaderTimeout: responseHeaderTimeout,
		clock:                 clock,
	}
}

func (r *Responder) RespondTo(
	ctx context.Context,
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
	responseWriter ResponseWriter,
) {
	upstreamResponse, err := r.upstream.RoundTrip(
		ctx,
		r.proxiedRequestFor(method, address, requestHeaders),
	)
	if err != nil {
		r.endFailedRoundTrip(ctx, address, responseWriter)
		return
	}
	defer func() { _ = upstreamResponse.Body.Close() }()
	r.send(ctx, address, upstreamResponse, responseWriter)
}

func (r *Responder) proxiedRequestFor(
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
) proxiedrequest.Request {
	return proxiedrequest.Request{
		Method:  method,
		Address: address,
		Headers: relayedheaders.RequestHeadersFrom(requestHeaders),
		HeadersDeadline: responsedeadline.HeadersDeadlineFrom(
			r.clock.Now(),
			requestHeaders.Values("Prefer"),
			r.responseHeaderTimeout,
		),
	}
}

func (r *Responder) endFailedRoundTrip(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	responseWriter ResponseWriter,
) {
	if ctx.Err() != nil {
		r.observers.ClientClosedRequest(ctx, address)
		return
	}
	responseWriter.SendHeaders(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (r *Responder) send(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	upstreamResponse *http.Response,
	responseWriter ResponseWriter,
) {
	responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		relayedheaders.ResponseHeadersFrom(upstreamResponse.Header),
	)
	clientWrites := &clientWriter{responseWriter: responseWriter}
	if _, err := io.Copy(clientWrites, upstreamResponse.Body); err != nil {
		r.observers.ResponseLeftIncomplete(
			ctx,
			address,
			incompleteResponseCauseOf(ctx, clientWrites.writeFailed, err),
			err,
		)
		responseWriter.Abort()
	}
}

func incompleteResponseCauseOf(
	ctx context.Context,
	clientWriteFailed bool,
	copyFailure error,
) IncompleteResponseCause {
	switch {
	case clientWriteFailed || ctx.Err() != nil:
		return ClientClosedRequest
	case errors.Is(copyFailure, roundtripcancel.ErrIdleTimeoutPassed):
		return RelayIdleTimeout
	default:
		return BodyRestReadFailed
	}
}
