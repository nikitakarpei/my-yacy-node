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

type relayer struct {
	egress          Egress
	pageAssessor    pageAssessor
	clock           Clock
	readingSlots    readingSlots
	pageByteCeiling int
	address         canonicalurl.CanonicalURL
	headersDueAt    time.Time
	timer           *readtimer.Timer
	replier         replier
}

func (r relayer) relayAnswerTo(ctx context.Context, egressRequest *http.Request) {
	r.timer.EndReadingAt(r.headersDueAt)
	answer, err := r.egress.RoundTrip(egressRequest)
	if err != nil {
		r.replier.failReading(ctx, PageReadDeadline, err)
		return
	}
	defer func() { _ = answer.Body.Close() }()
	r.relayAnswer(ctx, egressRequest, answer)
}

func (r relayer) relayAnswer(
	ctx context.Context,
	egressRequest *http.Request,
	answer *http.Response,
) {
	if r.clock.Now().After(r.headersDueAt) {
		r.replier.refuse(ctx, PageReadDeadline)
		return
	}
	if reason, skipped := skipReasonOf(egressRequest.Method, answer); skipped {
		r.replier.passThrough(ctx, reason, answer)
		return
	}
	if !contentencoding.IsDecodable(contentEncodingOf(answer.Header)) {
		r.replier.refuse(ctx, UndecodableEncoding)
		return
	}
	r.relayPage(ctx, egressRequest.Context(), answer)
}

func contentEncodingOf(headers http.Header) string {
	return strings.ToLower(strings.TrimSpace(headers.Get("Content-Encoding")))
}

func (r relayer) relayPage(
	ctx context.Context,
	readingCtx context.Context,
	answer *http.Response,
) {
	if !r.readingSlots.reserve(readingCtx) {
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

func (r relayer) bodyPrefixFrom(answerBody io.Reader) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(io.LimitReader(answerBody, int64(r.pageByteCeiling)))
	r.timer.Stop()
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress gave it
}
