package requestrelay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
)

type upstreamResponseRelayer struct {
	observers            Observers
	assessmentRunner     AssessmentRunner
	clock                Clock
	readingSlots         readingSlots
	pageByteCeiling      int
	method               string
	address              canonicalurl.CanonicalURL
	headersDueAt         time.Time
	stopPageReadDeadline func()
	responseSender       responseSender
}

func (r upstreamResponseRelayer) relay(
	ctx context.Context,
	readingCtx context.Context,
	upstreamResponse *http.Response,
) {
	if r.clock.Now().After(r.headersDueAt) {
		r.responseSender.refuse(ctx, PageReadDeadline)
		return
	}
	if reason, skipped := skipReasonOf(r.method, upstreamResponse); skipped {
		r.stopPageReadDeadline()
		r.responseSender.passThrough(readingCtx, reason, upstreamResponse)
		return
	}
	if !contentencoding.IsDecodable(contentEncodingOf(upstreamResponse.Header)) {
		r.responseSender.refuse(ctx, UndecodableEncoding)
		return
	}
	r.relayPage(ctx, readingCtx, upstreamResponse)
}

func contentEncodingOf(headers http.Header) string {
	return strings.ToLower(strings.TrimSpace(headers.Get("Content-Encoding")))
}

func (r upstreamResponseRelayer) relayPage(
	ctx context.Context,
	readingCtx context.Context,
	upstreamResponse *http.Response,
) {
	page, assessed := r.assessInReadingSlot(ctx, readingCtx, upstreamResponse)
	if !assessed {
		return
	}
	r.responseSender.sendPage(readingCtx, upstreamResponse, page)
}

func (r upstreamResponseRelayer) assessInReadingSlot(
	ctx context.Context,
	readingCtx context.Context,
	upstreamResponse *http.Response,
) (assessedPage, bool) {
	if !r.reserveReadingSlot(ctx, readingCtx) {
		r.responseSender.endCancelledReading(readingCtx, SlotWaitDeadline)
		return assessedPage{}, false
	}
	defer r.readingSlots.release()
	bodyPrefix, readFailure := r.bodyPrefixFrom(upstreamResponse.Body)
	r.stopPageReadDeadline()
	if readFailure != nil {
		r.responseSender.failReading(
			readingCtx,
			PageReadDeadline,
			BodyPrefixReadFailed,
			readFailure,
		)
		return assessedPage{}, false
	}
	page, refusal := r.assessedPageFrom(
		ctx, bodyPrefix, upstreamResponse.Header,
	)
	if refusal != "" {
		r.responseSender.refuse(ctx, refusal)
		return assessedPage{}, false
	}
	return page, true
}

func (r upstreamResponseRelayer) reserveReadingSlot(
	ctx context.Context,
	readingCtx context.Context,
) bool {
	slotWaitStarted := r.clock.Now()
	reserved := r.readingSlots.reserve(readingCtx)
	r.observers.ReadingSlotWaited(ctx, r.address, r.clock.Now().Sub(slotWaitStarted))
	return reserved
}

func (r upstreamResponseRelayer) bodyPrefixFrom(
	upstreamResponseBodyReader io.Reader,
) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(
		io.LimitReader(upstreamResponseBodyReader, int64(r.pageByteCeiling)),
	)
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress proxy gave it
}

func (r upstreamResponseRelayer) assessedPageFrom(
	ctx context.Context,
	bodyPrefix []byte,
	upstreamResponseHeaders http.Header,
) (assessedPage, RefusalReason) {
	body, decoded := contentencoding.DecodedBodyFrom(
		bodyPrefix,
		contentEncodingOf(upstreamResponseHeaders),
		r.pageByteCeiling,
	)
	if !decoded {
		return assessedPage{}, UndecodableBody
	}
	assessment, outcome := r.assessmentRunner.Assess(
		ctx,
		r.address,
		body,
		upstreamResponseHeaders,
		r.headersDueAt,
	)
	if outcome != assessmentgate.Assessed {
		return assessedPage{}, refusalsPerOutcome[outcome]
	}
	r.observers.PageAssessed(ctx, r.address, assessment)
	if !r.clock.Now().Before(r.headersDueAt) {
		return assessedPage{}, AssessmentDeadline
	}
	return assessedPage{
		bodyPrefix:   bodyPrefix,
		wasBodyWhole: len(bodyPrefix) < r.pageByteCeiling,
		assessment:   assessment,
	}, ""
}
