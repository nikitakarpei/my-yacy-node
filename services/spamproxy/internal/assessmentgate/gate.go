// Package assessmentgate runs page assessments on a bounded number of
// goroutines, and gives up on a page that misses its deadline.
package assessmentgate

import (
	"context"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

type PageAssessor interface {
	AssessmentFrom(
		address canonicalurl.CanonicalURL,
		body []byte,
		responseHeaders http.Header,
	) spamassessment.Assessment
}

type Outcome int

const (
	Assessed Outcome = iota
	SlotWaitDeadline
	AssessmentDeadline
	Panicked
)

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

type Gate struct {
	assessor        PageAssessor
	assessmentSlots chan struct{}
	clock           Clock
	observers       Observers
}

func New(assessor PageAssessor, slotAmount int, clock Clock, observers Observers) *Gate {
	return &Gate{
		assessor:        assessor,
		assessmentSlots: make(chan struct{}, slotAmount),
		clock:           clock,
		observers:       observers,
	}
}

func (g *Gate) AssessmentFrom(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	deadline time.Time,
) (spamassessment.Assessment, Outcome) {
	expired := make(chan struct{})
	stop := g.clock.After(deadline.Sub(g.clock.Now()), func() { close(expired) })
	defer stop()
	if !g.reserveSlot(ctx, address, expired) {
		return spamassessment.Assessment{}, SlotWaitDeadline
	}
	assessments := make(chan spamassessment.Assessment, 1)
	go g.assess(ctx, address, body, responseHeaders, assessments)
	select {
	case assessment, assessed := <-assessments:
		if !assessed {
			return spamassessment.Assessment{}, Panicked
		}
		return assessment, Assessed
	case <-expired:
		return spamassessment.Assessment{}, AssessmentDeadline
	}
}

func (g *Gate) reserveSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	expired <-chan struct{},
) bool {
	slotWaitStarted := g.clock.Now()
	defer func() { g.observers.SlotWaited(ctx, address, g.clock.Now().Sub(slotWaitStarted)) }()
	select {
	case g.assessmentSlots <- struct{}{}:
		return true
	case <-expired:
		return false
	}
}

func (g *Gate) assess(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	assessments chan<- spamassessment.Assessment,
) {
	defer func() { <-g.assessmentSlots }()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			g.observers.AssessmentPanicked(ctx, address, panicValue)
			close(assessments)
		}
	}()
	assessmentStarted := g.clock.Now()
	assessment := g.assessor.AssessmentFrom(address, body, responseHeaders)
	g.observers.AssessmentFinished(ctx, address, g.clock.Now().Sub(assessmentStarted))
	assessments <- assessment
}
