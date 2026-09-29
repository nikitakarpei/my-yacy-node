// Package pagerelay relays the answer of the egress proxy to the client, and
// adds a spam verdict to each page.
package pagerelay

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/readtimer"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/replydeadlines"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
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
	) (spamassessment.Assessment, bool)
}

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type Reply interface {
	io.Writer
	SendHead(status int, headers http.Header)
	CutShort()
}

type Limits struct {
	PageByteCeiling        int
	MaxPagesAssessedAtOnce int
	ReplyTimeouts          replydeadlines.Timeouts
	RelayIdleTimeout       time.Duration
}

type Relay struct {
	egress          Egress
	assessor        Assessor
	observers       Observers
	limits          Limits
	clock           Clock
	assessmentSlots assessmentSlots
}

func New(egress Egress, assessor Assessor, observers Observers, limits Limits, clock Clock) *Relay {
	return &Relay{
		egress:          egress,
		assessor:        assessor,
		observers:       observers,
		limits:          limits,
		clock:           clock,
		assessmentSlots: make(assessmentSlots, limits.MaxPagesAssessedAtOnce),
	}
}

func (r *Relay) ReplyTo(
	ctx context.Context,
	method string,
	address canonicalurl.CanonicalURL,
	requestHeaders http.Header,
	reply Reply,
) {
	requestArrivedAt := r.clock.Now()
	deadlines := replydeadlines.DeadlinesFrom(
		requestArrivedAt,
		requestHeaders.Values("Prefer"),
		r.limits.ReplyTimeouts,
	)
	readingCtx, cancelReading := context.WithCancel(ctx)
	defer cancelReading()
	pageExchange := r.exchangeFor(address, readtimer.New(r.clock, cancelReading), reply)
	defer pageExchange.timer.Stop()
	if !deadlines.ReadingEndsAt.After(requestArrivedAt) {
		pageExchange.refuse(ctx, WaitTooShort)
		return
	}
	pageExchange.relayFromEgress(
		ctx,
		egressRequestFor(readingCtx, method, address, requestHeaders),
		deadlines,
	)
}

func (r *Relay) exchangeFor(
	address canonicalurl.CanonicalURL,
	timer *readtimer.Timer,
	reply Reply,
) exchange {
	return exchange{
		egress:          r.egress,
		assessor:        r.assessor,
		observers:       r.observers,
		limits:          r.limits,
		clock:           r.clock,
		assessmentSlots: r.assessmentSlots,
		address:         address,
		timer:           timer,
		reply:           reply,
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
