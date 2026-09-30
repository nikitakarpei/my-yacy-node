// Package deadlineenforcing refuses a request whose response headers cannot
// arrive in time, and cancels the round trip at the headers deadline.
package deadlineenforcing

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxiedrequest"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtripcancel"
)

type Upstream interface {
	RoundTrip(ctx context.Context, request proxiedrequest.Request) (*http.Response, error)
}

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type RoundTripper struct {
	upstream  Upstream
	clock     Clock
	observers Observers
}

func New(upstream Upstream, clock Clock, observers Observers) *RoundTripper {
	return &RoundTripper{upstream: upstream, clock: clock, observers: observers}
}

func (r *RoundTripper) RoundTrip(
	ctx context.Context,
	request proxiedrequest.Request,
) (*http.Response, error) {
	if !request.HeadersDeadline.After(r.clock.Now()) {
		return r.refuse(ctx, request.Address, WaitTooShort), nil
	}
	roundTripCtx, cancelRoundTrip := context.WithCancelCause(ctx)
	upstreamResponse, err := r.roundTripBeforeDeadline(roundTripCtx, cancelRoundTrip, request)
	if err != nil {
		cancelRoundTrip(nil)
		return r.endFailedRoundTrip(ctx, roundTripCtx, request.Address, err)
	}
	return withRoundTripCancelledOnClose(upstreamResponse, cancelRoundTrip), nil
}

func (r *RoundTripper) refuse(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason RefusalReason,
) *http.Response {
	r.observers.RequestRefused(ctx, address, reason)
	return &http.Response{
		StatusCode: http.StatusGatewayTimeout,
		Header:     http.Header{"Content-Length": {"0"}},
		Body:       http.NoBody,
	}
}

func (r *RoundTripper) roundTripBeforeDeadline(
	roundTripCtx context.Context,
	cancelRoundTrip context.CancelCauseFunc,
	request proxiedrequest.Request,
) (*http.Response, error) {
	stopDeadline := r.clock.After(request.HeadersDeadline.Sub(r.clock.Now()), func() {
		cancelRoundTrip(roundtripcancel.ErrHeadersDeadlinePassed)
	})
	defer stopDeadline()
	return r.upstream.RoundTrip(roundTripCtx, request)
}

func (r *RoundTripper) endFailedRoundTrip(
	ctx context.Context,
	roundTripCtx context.Context,
	address canonicalurl.CanonicalURL,
	roundTripFailure error,
) (*http.Response, error) {
	if errors.Is(context.Cause(roundTripCtx), roundtripcancel.ErrHeadersDeadlinePassed) {
		return r.refuse(ctx, address, HeadersDeadline), nil
	}
	return nil, roundTripFailure
}
