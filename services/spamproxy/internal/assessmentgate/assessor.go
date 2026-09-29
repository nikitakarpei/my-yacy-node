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

type Assessor struct {
	pageAssessor    PageAssessor
	assessmentSlots chan struct{}
	clock           Clock
	observers       Observers
}

func New(pageAssessor PageAssessor, slotAmount int, clock Clock, observers Observers) *Assessor {
	return &Assessor{
		pageAssessor:    pageAssessor,
		assessmentSlots: make(chan struct{}, slotAmount),
		clock:           clock,
		observers:       observers,
	}
}

func (a *Assessor) Assess(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	deadline time.Time,
) (spamassessment.Assessment, Outcome) {
	expired := make(chan struct{})
	stop := a.clock.After(deadline.Sub(a.clock.Now()), func() { close(expired) })
	defer stop()
	if !a.reserveSlot(ctx, address, expired) {
		return spamassessment.Assessment{}, SlotWaitDeadline
	}
	assessments := make(chan spamassessment.Assessment, 1)
	go a.assessInSlot(ctx, address, body, responseHeaders, assessments)
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

func (a *Assessor) reserveSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	expired <-chan struct{},
) bool {
	slotWaitStarted := a.clock.Now()
	defer func() { a.observers.SlotWaited(ctx, address, a.clock.Now().Sub(slotWaitStarted)) }()
	select {
	case a.assessmentSlots <- struct{}{}:
		return true
	case <-expired:
		return false
	}
}

func (a *Assessor) assessInSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	assessments chan<- spamassessment.Assessment,
) {
	defer func() { <-a.assessmentSlots }()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			a.observers.AssessmentPanicked(ctx, address, panicValue)
			close(assessments)
		}
	}()
	assessmentStarted := a.clock.Now()
	assessment := a.pageAssessor.AssessmentFrom(address, body, responseHeaders)
	a.observers.AssessmentFinished(ctx, address, a.clock.Now().Sub(assessmentStarted))
	assessments <- assessment
}
