package requestrelay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
)

type upstreamResponseRelayer struct {
	observers       Observers
	pageAssessor    pageAssessor
	clock           Clock
	readingSlots    readingSlots
	pageByteCeiling int
	method          string
	address         canonicalurl.CanonicalURL
	headersDueAt    time.Time
	canceller       *readingcancel.Canceller
	responseSender  responseSender
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
		r.responseSender.passThrough(ctx, reason, upstreamResponse)
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
	r.responseSender.sendPage(ctx, upstreamResponse, page)
}

func (r upstreamResponseRelayer) assessInReadingSlot(
	ctx context.Context,
	readingCtx context.Context,
	upstreamResponse *http.Response,
) (assessedPage, bool) {
	if !r.reserveReadingSlot(ctx, readingCtx) {
		r.responseSender.endCancelledReading(ctx, SlotWaitDeadline)
		return assessedPage{}, false
	}
	defer r.readingSlots.release()
	bodyPrefix, readFailure := r.bodyPrefixFrom(upstreamResponse.Body)
	if readFailure != nil {
		r.responseSender.failReading(ctx, PageReadDeadline, BodyPrefixReadFailed, readFailure)
		return assessedPage{}, false
	}
	page, refusal := r.pageAssessor.assess(
		ctx, r.address, bodyPrefix, upstreamResponse.Header, r.headersDueAt,
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
	r.canceller.Stop()
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress proxy gave it
}
