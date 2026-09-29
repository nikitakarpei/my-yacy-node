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

type replier struct {
	replyWriter      ReplyWriter
	observers        Observers
	address          canonicalurl.CanonicalURL
	canceller        *readingcancel.Canceller
	relayIdleTimeout time.Duration
}

func (c replier) refuse(ctx context.Context, reason RefusalReason) {
	c.observers.RequestRefused(ctx, c.address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	c.replyWriter.SendHead(httpStatusesPerRefusal[reason], headers)
}

func (c replier) failReading(ctx context.Context, expiryReason RefusalReason, cause error) {
	if c.canceller.Cancelled() {
		c.refuse(ctx, expiryReason)
		return
	}
	c.reportReadingFailure(ctx, cause)
	c.replyWriter.SendHead(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (c replier) reportReadingFailure(ctx context.Context, cause error) {
	if ctx.Err() != nil {
		c.observers.ClientLeft(ctx, c.address)
		return
	}
	c.observers.AnswerReadingFailed(ctx, c.address, cause)
}

func (c replier) passThrough(ctx context.Context, reason SkipReason, answer *http.Response) {
	c.observers.AssessmentSkipped(ctx, c.address, reason)
	c.replyWriter.SendHead(answer.StatusCode, relayedheaders.EndToEndHeadersOf(answer.Header))
	c.relayRest(ctx, answer.Body, nil)
}

func (c replier) sendPage(ctx context.Context, answer *http.Response, page assessedPage) {
	c.replyWriter.SendHead(answer.StatusCode, page.headersFrom(answer.Header))
	c.relayRest(ctx, answer.Body, page.bodyPrefix)
}

func (c replier) relayRest(ctx context.Context, answerBody io.Reader, bodyPrefix []byte) {
	rest := c.canceller.IdleLimitedFrom(answerBody, c.relayIdleTimeout)
	if _, err := io.Copy(
		c.replyWriter,
		io.MultiReader(bytes.NewReader(bodyPrefix), rest),
	); err != nil {
		c.reportCutShort(ctx, err)
		c.replyWriter.CutShort()
	}
}

func (c replier) reportCutShort(ctx context.Context, cause error) {
	if ctx.Err() != nil {
		c.observers.ClientLeft(ctx, c.address)
		return
	}
	c.observers.ReplyCutShort(ctx, c.address, cause)
}
