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

type Assessor interface {
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

type Runner struct {
	assessor        Assessor
	assessmentSlots chan struct{}
	clock           Clock
	observers       Observers
}

func New(assessor Assessor, slotAmount int, clock Clock, observers Observers) *Runner {
	return &Runner{
		assessor:        assessor,
		assessmentSlots: make(chan struct{}, slotAmount),
		clock:           clock,
		observers:       observers,
	}
}

func (r *Runner) Assess(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	deadline time.Time,
) (spamassessment.Assessment, Outcome) {
	expired := make(chan struct{})
	stop := r.clock.After(deadline.Sub(r.clock.Now()), func() { close(expired) })
	defer stop()
	if !r.reserveSlot(ctx, address, expired) {
		return spamassessment.Assessment{}, SlotWaitDeadline
	}
	assessments := make(chan spamassessment.Assessment, 1)
	go r.assessInSlot(ctx, address, body, responseHeaders, assessments)
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

func (r *Runner) reserveSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	expired <-chan struct{},
) bool {
	slotWaitStarted := r.clock.Now()
	defer func() { r.observers.SlotWaited(ctx, address, r.clock.Now().Sub(slotWaitStarted)) }()
	select {
	case r.assessmentSlots <- struct{}{}:
		return true
	case <-expired:
		return false
	}
}

func (r *Runner) assessInSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	assessments chan<- spamassessment.Assessment,
) {
	defer func() { <-r.assessmentSlots }()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			r.observers.AssessmentPanicked(ctx, address, panicValue)
			close(assessments)
		}
	}()
	assessmentStarted := r.clock.Now()
	assessment := r.assessor.AssessmentFrom(address, body, responseHeaders)
	r.observers.AssessmentFinished(ctx, address, len(body), r.clock.Now().Sub(assessmentStarted))
	assessments <- assessment
}
