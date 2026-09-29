package requestrelay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readtimer"
)

type answerRelayer struct {
	observers       Observers
	pageAssessor    pageAssessor
	clock           Clock
	readingSlots    readingSlots
	pageByteCeiling int
	method          string
	address         canonicalurl.CanonicalURL
	headersDueAt    time.Time
	timer           *readtimer.Timer
	replier         replier
}

func (r answerRelayer) relay(
	ctx context.Context,
	readingCtx context.Context,
	answer *http.Response,
) {
	if r.clock.Now().After(r.headersDueAt) {
		r.replier.refuse(ctx, PageReadDeadline)
		return
	}
	if reason, skipped := skipReasonOf(r.method, answer); skipped {
		r.replier.passThrough(ctx, reason, answer)
		return
	}
	if !contentencoding.IsDecodable(contentEncodingOf(answer.Header)) {
		r.replier.refuse(ctx, UndecodableEncoding)
		return
	}
	r.relayPage(ctx, readingCtx, answer)
}

func contentEncodingOf(headers http.Header) string {
	return strings.ToLower(strings.TrimSpace(headers.Get("Content-Encoding")))
}

func (r answerRelayer) relayPage(
	ctx context.Context,
	readingCtx context.Context,
	answer *http.Response,
) {
	if !r.reserveReadingSlot(ctx, readingCtx) {
		r.replier.failReading(ctx, SlotWaitDeadline, context.Cause(readingCtx))
		return
	}
	bodyPrefix, readFailure := r.bodyPrefixFrom(answer.Body)
	if readFailure != nil {
		r.readingSlots.release()
		r.replier.failReading(ctx, PageReadDeadline, readFailure)
		return
	}
	page, refusal := r.pageAssessor.assessedPageFrom(
		ctx, r.address, bodyPrefix, answer.Header, r.headersDueAt,
	)
	r.readingSlots.release()
	if refusal != "" {
		r.replier.refuse(ctx, refusal)
		return
	}
	r.replier.sendPage(ctx, answer, page)
}

func (r answerRelayer) reserveReadingSlot(ctx context.Context, readingCtx context.Context) bool {
	slotWaitStarted := r.clock.Now()
	reserved := r.readingSlots.reserve(readingCtx)
	r.observers.ReadingSlotWaited(ctx, r.address, r.clock.Now().Sub(slotWaitStarted))
	return reserved
}

func (r answerRelayer) bodyPrefixFrom(answerBody io.Reader) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(io.LimitReader(answerBody, int64(r.pageByteCeiling)))
	r.timer.Stop()
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress gave it
}
