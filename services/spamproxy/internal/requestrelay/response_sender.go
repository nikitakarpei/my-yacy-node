package requestrelay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
)

const retryAfterSeconds = "1"

type responseSender struct {
	responseWriter   ResponseWriter
	observers        Observers
	clock            Clock
	address          canonicalurl.CanonicalURL
	cancelReading    context.CancelCauseFunc
	relayIdleTimeout time.Duration
}

func (s responseSender) refuse(ctx context.Context, reason RefusalReason) {
	s.observers.RequestRefused(ctx, s.address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	s.responseWriter.SendHeaders(httpStatusesPerRefusal[reason], headers)
}

func (s responseSender) failReading(
	readingCtx context.Context,
	expiryReason RefusalReason,
	failure UpstreamResponseFailure,
	cause error,
) {
	if readingCtx.Err() != nil {
		s.endCancelledReading(readingCtx, expiryReason)
		return
	}
	s.observers.UpstreamResponseFailed(readingCtx, s.address, failure, cause)
	s.responseWriter.SendHeaders(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (s responseSender) endCancelledReading(
	readingCtx context.Context,
	expiryReason RefusalReason,
) {
	if errors.Is(context.Cause(readingCtx), readingcancel.ErrDeadlinePassed) {
		s.refuse(readingCtx, expiryReason)
		return
	}
	s.observers.ClientClosedRequest(readingCtx, s.address)
}

func (s responseSender) passThrough(
	readingCtx context.Context,
	reason SkipReason,
	upstreamResponse *http.Response,
) {
	s.observers.AssessmentSkipped(readingCtx, s.address, reason)
	s.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		relayedheaders.ResponseHeadersFrom(upstreamResponse.Header),
	)
	s.relayRest(readingCtx, upstreamResponse.Body, nil)
}

func (s responseSender) sendPage(
	readingCtx context.Context,
	upstreamResponse *http.Response,
	page assessedPage,
) {
	s.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		page.headersFrom(upstreamResponse.Header),
	)
	s.relayRest(readingCtx, upstreamResponse.Body, page.bodyPrefix)
}

func (s responseSender) relayRest(
	readingCtx context.Context,
	upstreamResponseBodyReader io.Reader,
	bodyPrefix []byte,
) {
	unreadBodyReader := readingcancel.IdleCancellingReaderFrom(
		upstreamResponseBodyReader,
		s.clock,
		s.cancelReading,
		s.relayIdleTimeout,
	)
	clientWrites := &clientWriter{responseWriter: s.responseWriter}
	if _, err := io.Copy(
		clientWrites,
		io.MultiReader(bytes.NewReader(bodyPrefix), unreadBodyReader),
	); err != nil {
		s.observers.ResponseLeftIncomplete(
			readingCtx,
			s.address,
			incompleteResponseCauseOf(readingCtx, clientWrites.writeFailed),
			err,
		)
		s.responseWriter.Abort()
	}
}

func incompleteResponseCauseOf(
	readingCtx context.Context,
	clientWriteFailed bool,
) IncompleteResponseCause {
	readingCancelCause := context.Cause(readingCtx)
	switch {
	case clientWriteFailed || errors.Is(readingCancelCause, context.Canceled):
		return ClientClosedRequest
	case errors.Is(readingCancelCause, readingcancel.ErrIdleTimeoutPassed):
		return RelayIdleTimeout
	default:
		return BodyRestReadFailed
	}
}

type clientWriter struct {
	responseWriter ResponseWriter
	writeFailed    bool
}

func (w *clientWriter) Write(chunk []byte) (int, error) {
	written, err := w.responseWriter.Write(chunk)
	w.writeFailed = err != nil
	return written, err //nolint:wrapcheck // io.Copy hands the cause of the client to the relay
}
