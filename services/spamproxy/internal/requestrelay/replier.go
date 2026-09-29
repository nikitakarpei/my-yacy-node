package requestrelay

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readtimer"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
)

const retryAfterSeconds = "1"

type replier struct {
	reply            Reply
	observers        Observers
	address          canonicalurl.CanonicalURL
	timer            *readtimer.Timer
	relayIdleTimeout time.Duration
}

func (c replier) refuse(ctx context.Context, reason RefusalReason) {
	c.observers.RequestRefused(ctx, c.address, reason)
	headers := http.Header{"Content-Length": {"0"}}
	if reason.isRetryable() {
		headers.Set("Retry-After", retryAfterSeconds)
	}
	c.reply.SendHead(httpStatusesPerRefusal[reason], headers)
}

func (c replier) failReading(ctx context.Context, expiryReason RefusalReason, cause error) {
	if c.timer.Expired() {
		c.refuse(ctx, expiryReason)
		return
	}
	c.observers.AnswerReadingFailed(ctx, c.address, cause)
	c.reply.SendHead(http.StatusBadGateway, http.Header{"Content-Length": {"0"}})
}

func (c replier) passThrough(ctx context.Context, reason SkipReason, answer *http.Response) {
	c.observers.AssessmentSkipped(ctx, c.address, reason)
	c.reply.SendHead(answer.StatusCode, relayedheaders.EndToEndHeadersOf(answer.Header))
	c.relayRest(ctx, answer.Body, nil)
}

func (c replier) sendPage(ctx context.Context, answer *http.Response, page assessedPage) {
	c.reply.SendHead(answer.StatusCode, page.headersFrom(answer.Header))
	c.relayRest(ctx, answer.Body, page.bodyPrefix)
}

func (c replier) relayRest(ctx context.Context, answerBody io.Reader, bodyPrefix []byte) {
	rest := c.timer.IdleLimitedFrom(answerBody, c.relayIdleTimeout)
	if _, err := io.Copy(c.reply, io.MultiReader(bytes.NewReader(bodyPrefix), rest)); err != nil {
		c.observers.ReplyCutShort(ctx, c.address, err)
		c.reply.CutShort()
	}
}
