package requestrelay

import (
	"bytes"
	"context"
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
	address          canonicalurl.CanonicalURL
	canceller        *readingcancel.Canceller
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
	ctx context.Context,
	expiryReason RefusalReason,
	failure UpstreamResponseFailure,
	cause error,
) {
	if s.canceller.Cancelled() || ctx.Err() != nil {
		s.endCancelledReading(ctx, expiryReason)
		return
	}
	s.observers.UpstreamResponseFailed(ctx, s.address, failure, cause)
	s.responseWriter.SendHeaders(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (s responseSender) endCancelledReading(ctx context.Context, expiryReason RefusalReason) {
	if s.canceller.Cancelled() {
		s.refuse(ctx, expiryReason)
		return
	}
	s.observers.ClientClosedRequest(ctx, s.address)
}

func (s responseSender) passThrough(
	ctx context.Context,
	reason SkipReason,
	upstreamResponse *http.Response,
) {
	s.observers.AssessmentSkipped(ctx, s.address, reason)
	s.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		relayedheaders.EndToEndHeadersOf(upstreamResponse.Header),
	)
	s.relayRest(ctx, upstreamResponse.Body, nil)
}

func (s responseSender) sendPage(
	ctx context.Context,
	upstreamResponse *http.Response,
	page assessedPage,
) {
	s.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		page.headersFrom(upstreamResponse.Header),
	)
	s.relayRest(ctx, upstreamResponse.Body, page.bodyPrefix)
}

func (s responseSender) relayRest(
	ctx context.Context,
	upstreamResponseBodyReader io.Reader,
	bodyPrefix []byte,
) {
	unreadBodyReader := readingcancel.IdleCancellingReaderFrom(
		upstreamResponseBodyReader,
		s.canceller,
		s.relayIdleTimeout,
	)
	clientWrites := &clientWriter{responseWriter: s.responseWriter}
	if _, err := io.Copy(
		clientWrites,
		io.MultiReader(bytes.NewReader(bodyPrefix), unreadBodyReader),
	); err != nil {
		s.observers.ResponseLeftIncomplete(
			ctx,
			s.address,
			s.incompleteResponseCauseOf(ctx, clientWrites.writeFailed),
			err,
		)
		s.responseWriter.Abort()
	}
}

func (s responseSender) incompleteResponseCauseOf(
	ctx context.Context,
	clientWriteFailed bool,
) IncompleteResponseCause {
	switch {
	case clientWriteFailed || ctx.Err() != nil:
		return ClientClosedRequest
	case s.canceller.Cancelled():
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
