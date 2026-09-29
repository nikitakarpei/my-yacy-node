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
	responder       responder
}

func (r upstreamResponseRelayer) relay(
	ctx context.Context,
	readingCtx context.Context,
	upstreamResponse *http.Response,
) {
	if r.clock.Now().After(r.headersDueAt) {
		r.responder.refuse(ctx, PageReadDeadline)
		return
	}
	if reason, skipped := skipReasonOf(r.method, upstreamResponse); skipped {
		r.responder.passThrough(ctx, reason, upstreamResponse)
		return
	}
	if !contentencoding.IsDecodable(contentEncodingOf(upstreamResponse.Header)) {
		r.responder.refuse(ctx, UndecodableEncoding)
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
	if !r.reserveReadingSlot(ctx, readingCtx) {
		r.responder.endCancelledReading(ctx, SlotWaitDeadline)
		return
	}
	bodyPrefix, readFailure := r.bodyPrefixFrom(upstreamResponse.Body)
	if readFailure != nil {
		r.readingSlots.release()
		r.responder.failReading(ctx, PageReadDeadline, BodyPrefixReadFailed, readFailure)
		return
	}
	page, refusal := r.pageAssessor.assess(
		ctx, r.address, bodyPrefix, upstreamResponse.Header, r.headersDueAt,
	)
	r.readingSlots.release()
	if refusal != "" {
		r.responder.refuse(ctx, refusal)
		return
	}
	r.responder.sendPage(ctx, upstreamResponse, page)
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

func (r upstreamResponseRelayer) bodyPrefixFrom(upstreamResponseBody io.Reader) ([]byte, error) {
	bodyPrefix, err := io.ReadAll(io.LimitReader(upstreamResponseBody, int64(r.pageByteCeiling)))
	r.canceller.Stop()
	return bodyPrefix, err //nolint:wrapcheck // the relay reports the cause as the egress proxy gave it
}
