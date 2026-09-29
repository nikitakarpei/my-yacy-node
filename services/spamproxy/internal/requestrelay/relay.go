// Package requestrelay relays the upstream response of the egress proxy to the
// client, and adds a spam verdict to each page.
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
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/responsedeadline"
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

type ResponseWriter interface {
	io.Writer
	SendHeaders(status int, headers http.Header)
	Abort()
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

func (r *Relayer) RespondTo(
	ctx context.Context,
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
	responseWriter ResponseWriter,
) {
	requestArrivedAt := r.clock.Now()
	headersDueAt := responsedeadline.HeadersDueAtFrom(
		requestArrivedAt,
		requestHeaders.Values("Prefer"),
		r.limits.ResponseHeaderTimeout,
	)
	readingCtx, cancelReading := context.WithCancel(ctx)
	defer cancelReading()
	canceller := readingcancel.New(r.clock, cancelReading)
	defer canceller.Stop()
	responder := r.responderFor(address, canceller, responseWriter)
	if !headersDueAt.After(requestArrivedAt) {
		responder.refuse(ctx, WaitTooShort)
		return
	}
	upstreamRequest := upstreamRequestFor(readingCtx, method, address, requestHeaders)
	canceller.CancelAt(headersDueAt)
	upstreamResponse, err := r.egress.RoundTrip(upstreamRequest)
	if err != nil {
		responder.failReading(ctx, PageReadDeadline, NoResponse, err)
		return
	}
	defer func() { _ = upstreamResponse.Body.Close() }()
	r.upstreamResponseRelayerFor(method, address, headersDueAt, canceller, responder).
		relay(ctx, readingCtx, upstreamResponse)
}

func (r *Relayer) responderFor(
	address canonicalurl.CanonicalURL,
	canceller *readingcancel.Canceller,
	responseWriter ResponseWriter,
) responder {
	return responder{
		responseWriter:   responseWriter,
		observers:        r.observers,
		address:          address,
		canceller:        canceller,
		relayIdleTimeout: r.limits.RelayIdleTimeout,
	}
}

func (r *Relayer) upstreamResponseRelayerFor(
	method string,
	address canonicalurl.CanonicalURL,
	headersDueAt time.Time,
	canceller *readingcancel.Canceller,
	responder responder,
) upstreamResponseRelayer {
	return upstreamResponseRelayer{
		observers:       r.observers,
		pageAssessor:    r.pageAssessor,
		clock:           r.clock,
		readingSlots:    r.readingSlots,
		pageByteCeiling: r.limits.PageByteCeiling,
		method:          method,
		address:         address,
		headersDueAt:    headersDueAt,
		canceller:       canceller,
		responder:       responder,
	}
}

func upstreamRequestFor(
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
