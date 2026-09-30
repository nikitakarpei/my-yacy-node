// Package idlecancelling cancels reading an upstream response body that waits
// too long for its next bytes.
package idlecancelling

import (
	"context"
	"net/http"
	"time"
)

type Upstream interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type RoundTripper struct {
	upstream    Upstream
	clock       Clock
	idleTimeout time.Duration
}

func New(upstream Upstream, idleTimeout time.Duration, clock Clock) *RoundTripper {
	return &RoundTripper{upstream: upstream, clock: clock, idleTimeout: idleTimeout}
}

func (r *RoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	readingCtx, cancelReading := context.WithCancelCause(request.Context())
	upstreamResponse, err := r.upstream.RoundTrip(request.WithContext(readingCtx))
	if err != nil {
		cancelReading(nil)
		return nil, err
	}
	upstreamResponse.Body = idleCancellingBody{
		body:          upstreamResponse.Body,
		clock:         r.clock,
		idleTimeout:   r.idleTimeout,
		readingCtx:    readingCtx,
		cancelReading: cancelReading,
	}
	return upstreamResponse, nil
}
