// Package requestrelay relays the answer of the egress proxy to the client, and
// adds a spam verdict to each page.
package requestrelay

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readingcancel"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadline"
)

type Egress interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type Assessor interface {
	Assess(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		body []byte,
		responseHeaders http.Header,
		deadline time.Time,
	) (spamassessment.Assessment, assessmentgate.Outcome)
}

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type ReplyWriter interface {
	io.Writer
	SendHead(status int, headers http.Header)
	CutShort()
}

type Limits struct {
	PageByteCeiling       int
	MaxPagesReadAtOnce    int
	ResponseHeaderTimeout time.Duration
	RelayIdleTimeout      time.Duration
}

type Relayer struct {
	egress       Egress
	pageAssessor pageAssessor
	observers    Observers
	limits       Limits
	clock        Clock
	readingSlots readingSlots
}

func New(
	egress Egress,
	assessor Assessor,
	observers Observers,
	limits Limits,
	clock Clock,
) *Relayer {
	return &Relayer{
		egress: egress,
		pageAssessor: pageAssessor{
			assessor:        assessor,
			observers:       observers,
			clock:           clock,
			pageByteCeiling: limits.PageByteCeiling,
		},
		observers:    observers,
		limits:       limits,
		clock:        clock,
		readingSlots: make(readingSlots, limits.MaxPagesReadAtOnce),
	}
}

func (r *Relayer) ReplyTo(
	ctx context.Context,
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
	replyWriter ReplyWriter,
) {
	requestArrivedAt := r.clock.Now()
	headersDueAt := replydeadline.HeadersDueAtFrom(
		requestArrivedAt,
		requestHeaders.Values("Prefer"),
		r.limits.ResponseHeaderTimeout,
	)
	readingCtx, cancelReading := context.WithCancel(ctx)
	defer cancelReading()
	canceller := readingcancel.New(r.clock, cancelReading)
	defer canceller.Stop()
	replier := r.replierFor(address, requestArrivedAt, canceller, replyWriter)
	if !headersDueAt.After(requestArrivedAt) {
		replier.refuse(ctx, WaitTooShort)
		return
	}
	egressRequest := egressRequestFor(readingCtx, method, address, requestHeaders)
	canceller.CancelAt(headersDueAt)
	answer, err := r.egress.RoundTrip(egressRequest)
	if err != nil {
		replier.failReading(ctx, PageReadDeadline, err)
		return
	}
	defer func() { _ = answer.Body.Close() }()
	r.answerRelayerFor(method, address, headersDueAt, canceller, replier).
		relay(ctx, readingCtx, answer)
}

func (r *Relayer) replierFor(
	address canonicalurl.CanonicalURL,
	requestArrivedAt time.Time,
	canceller *readingcancel.Canceller,
	replyWriter ReplyWriter,
) replier {
	return replier{
		replyWriter:      replyWriter,
		observers:        r.observers,
		clock:            r.clock,
		address:          address,
		requestArrivedAt: requestArrivedAt,
		canceller:        canceller,
		relayIdleTimeout: r.limits.RelayIdleTimeout,
	}
}

func (r *Relayer) answerRelayerFor(
	method string,
	address canonicalurl.CanonicalURL,
	headersDueAt time.Time,
	canceller *readingcancel.Canceller,
	replier replier,
) answerRelayer {
	return answerRelayer{
		observers:       r.observers,
		pageAssessor:    r.pageAssessor,
		clock:           r.clock,
		readingSlots:    r.readingSlots,
		pageByteCeiling: r.limits.PageByteCeiling,
		method:          method,
		address:         address,
		headersDueAt:    headersDueAt,
		canceller:       canceller,
		replier:         replier,
	}
}

func egressRequestFor(
	ctx context.Context,
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
) *http.Request {
	request := &http.Request{
		Method: method,
		URL:    address.WebAddress(),
		Header: relayedheaders.ForwardedHeadersFrom(requestHeaders),
	}
	return request.WithContext(ctx)
}
