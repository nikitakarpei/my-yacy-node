package deadlineenforcing

import (
	"context"
	"io"
	"net/http"
)

func withRoundTripCancelledOnClose(
	upstreamResponse *http.Response,
	cancelRoundTrip context.CancelCauseFunc,
) *http.Response {
	upstreamResponse.Body = roundTripCancellingBody{
		ReadCloser:      upstreamResponse.Body,
		cancelRoundTrip: cancelRoundTrip,
	}
	return upstreamResponse
}

type roundTripCancellingBody struct {
	io.ReadCloser
	cancelRoundTrip context.CancelCauseFunc
}

func (b roundTripCancellingBody) Close() error {
	defer b.cancelRoundTrip(nil)
	return b.ReadCloser.Close() //nolint:wrapcheck // the caller of Close gets the cause of the body
}
