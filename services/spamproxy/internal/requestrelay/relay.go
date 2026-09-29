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
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readtimer"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadline"
)

type Egress interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type Assessor interface {
	AssessmentFrom(
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

type Relay struct {
	egress       Egress
	pageAssessor pageAssessor
	observers    Observers
	limits       Limits
	clock        Clock
	readingSlots readingSlots
}

func New(egress Egress, assessor Assessor, observers Observers, limits Limits, clock Clock) *Relay {
	return &Relay{
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

func (r *Relay) ReplyTo(
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
	timer := readtimer.New(r.clock, cancelReading)
	defer timer.Stop()
	replier := r.replierFor(address, requestArrivedAt, timer, replyWriter)
	if !headersDueAt.After(requestArrivedAt) {
		replier.refuse(ctx, WaitTooShort)
		return
	}
	r.relayerFor(address, headersDueAt, timer, replier).relayAnswerTo(
		ctx,
		egressRequestFor(readingCtx, method, address, requestHeaders),
	)
}

func (r *Relay) replierFor(
	address canonicalurl.CanonicalURL,
	requestArrivedAt time.Time,
	timer *readtimer.Timer,
	replyWriter ReplyWriter,
) replier {
	return replier{
		replyWriter:      replyWriter,
		observers:        r.observers,
		clock:            r.clock,
		address:          address,
		requestArrivedAt: requestArrivedAt,
		timer:            timer,
		relayIdleTimeout: r.limits.RelayIdleTimeout,
	}
}

func (r *Relay) relayerFor(
	address canonicalurl.CanonicalURL,
	headersDueAt time.Time,
	timer *readtimer.Timer,
	replier replier,
) relayer {
	return relayer{
		egress:          r.egress,
		observers:       r.observers,
		pageAssessor:    r.pageAssessor,
		clock:           r.clock,
		readingSlots:    r.readingSlots,
		pageByteCeiling: r.limits.PageByteCeiling,
		address:         address,
		headersDueAt:    headersDueAt,
		timer:           timer,
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
