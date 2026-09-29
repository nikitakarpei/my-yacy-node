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

type responder struct {
	responseWriter   ResponseWriter
	observers        Observers
	address          canonicalurl.CanonicalURL
	canceller        *readingcancel.Canceller
	relayIdleTimeout time.Duration
}

func (c responder) refuse(ctx context.Context, reason RefusalReason) {
	c.observers.RequestRefused(ctx, c.address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	c.responseWriter.SendHeaders(httpStatusesPerRefusal[reason], headers)
}

func (c responder) failReading(
	ctx context.Context,
	expiryReason RefusalReason,
	failure UpstreamResponseFailure,
	cause error,
) {
	if c.canceller.Cancelled() || ctx.Err() != nil {
		c.endCancelledReading(ctx, expiryReason)
		return
	}
	c.observers.UpstreamResponseFailed(ctx, c.address, failure, cause)
	c.responseWriter.SendHeaders(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (c responder) endCancelledReading(ctx context.Context, expiryReason RefusalReason) {
	if c.canceller.Cancelled() {
		c.refuse(ctx, expiryReason)
		return
	}
	c.observers.ClientClosedRequest(ctx, c.address)
}

func (c responder) passThrough(
	ctx context.Context,
	reason SkipReason,
	upstreamResponse *http.Response,
) {
	c.observers.AssessmentSkipped(ctx, c.address, reason)
	c.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		relayedheaders.EndToEndHeadersOf(upstreamResponse.Header),
	)
	c.relayRest(ctx, upstreamResponse.Body, nil)
}

func (c responder) sendPage(
	ctx context.Context,
	upstreamResponse *http.Response,
	page assessedPage,
) {
	c.responseWriter.SendHeaders(
		upstreamResponse.StatusCode,
		page.headersFrom(upstreamResponse.Header),
	)
	c.relayRest(ctx, upstreamResponse.Body, page.bodyPrefix)
}

func (c responder) relayRest(
	ctx context.Context,
	upstreamResponseBody io.Reader,
	bodyPrefix []byte,
) {
	rest := c.canceller.IdleLimitedFrom(upstreamResponseBody, c.relayIdleTimeout)
	clientWrites := &clientWriter{responseWriter: c.responseWriter}
	if _, err := io.Copy(
		clientWrites,
		io.MultiReader(bytes.NewReader(bodyPrefix), rest),
	); err != nil {
		c.observers.ResponseLeftIncomplete(
			ctx,
			c.address,
			c.incompleteResponseCauseOf(ctx, clientWrites.writeFailed),
			err,
		)
		c.responseWriter.Abort()
	}
}

func (c responder) incompleteResponseCauseOf(
	ctx context.Context,
	clientWriteFailed bool,
) IncompleteResponseCause {
	switch {
	case clientWriteFailed || ctx.Err() != nil:
		return ClientClosedRequest
	case c.canceller.Cancelled():
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
