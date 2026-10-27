// Package classificationgate runs page classifications on a bounded number of
// goroutines, and gives up on a page when its round trip is cancelled.
package classificationgate

import (
	"context"
	"net/http"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

type Classifier interface {
	ClassificationOf(
		address canonicalurl.CanonicalURL,
		body []byte,
		responseHeaders http.Header,
	) pageclassification.Classification
}

type Outcome int

const (
	Classified Outcome = iota
	SlotWaitCancelled
	ClassificationCancelled
	Panicked
)

type Clock interface {
	Now() time.Time
}

type Runner struct {
	classifier          Classifier
	classificationSlots chan struct{}
	clock               Clock
	observers           Observers
}

func New(classifier Classifier, slotAmount int, clock Clock, observers Observers) *Runner {
	return &Runner{
		classifier:          classifier,
		classificationSlots: make(chan struct{}, slotAmount),
		clock:               clock,
		observers:           observers,
	}
}

func (r *Runner) Classify(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
) (pageclassification.Classification, Outcome) {
	if !r.reserveSlot(ctx, address) {
		return pageclassification.Classification{}, SlotWaitCancelled
	}
	classifications := make(chan pageclassification.Classification, 1)
	go r.classifyInSlot(ctx, address, body, responseHeaders, classifications)
	select {
	case classification, classified := <-classifications:
		if !classified {
			return pageclassification.Classification{}, Panicked
		}
		return classification, Classified
	case <-ctx.Done():
		return pageclassification.Classification{}, ClassificationCancelled
	}
}

func (r *Runner) reserveSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
) bool {
	slotWaitStarted := r.clock.Now()
	defer func() { r.observers.SlotWaited(ctx, address, r.clock.Now().Sub(slotWaitStarted)) }()
	select {
	case r.classificationSlots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *Runner) classifyInSlot(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
	classifications chan<- pageclassification.Classification,
) {
	defer func() { <-r.classificationSlots }()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			r.observers.ClassificationPanicked(ctx, address, panicValue)
			close(classifications)
		}
	}()
	classificationStarted := r.clock.Now()
	classification := r.classifier.ClassificationOf(address, body, responseHeaders)
	r.observers.ClassificationFinished(
		ctx,
		address,
		len(body),
		r.clock.Now().Sub(classificationStarted),
	)
	classifications <- classification
}
