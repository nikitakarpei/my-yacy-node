package assessmentgate_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

var (
	now        = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	deadline   = now.Add(time.Second)
	assessment = spamassessment.Assessment{Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"}
)

type result struct {
	assessment spamassessment.Assessment
	outcome    assessmentgate.Outcome
}

type fakeClock struct {
	expirations chan func()
}

func newFakeClock() fakeClock {
	return fakeClock{expirations: make(chan func(), 4)}
}

func (c fakeClock) Now() time.Time { return now }

func (c fakeClock) After(_ time.Duration, expire func()) func() {
	c.expirations <- expire
	return func() {}
}

type heldAssessor struct {
	started  chan struct{}
	released chan struct{}
}

func newHeldAssessor() heldAssessor {
	return heldAssessor{started: make(chan struct{}, 2), released: make(chan struct{})}
}

func (a heldAssessor) AssessmentFrom(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) spamassessment.Assessment {
	a.started <- struct{}{}
	<-a.released
	return assessment
}

type fixedAssessor struct{}

func (fixedAssessor) AssessmentFrom(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) spamassessment.Assessment {
	return assessment
}

type panickingAssessor struct{}

func (panickingAssessor) AssessmentFrom(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) spamassessment.Assessment {
	panic("broken page")
}

type panicRecord struct {
	mutex       sync.Mutex
	panicValues []any
}

func (r *panicRecord) AssessmentPanicked(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	panicValue any,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.panicValues = append(r.panicValues, panicValue)
}

func TestAnAssessmentWithinTheDeadlineIsTheAssessmentOfTheAssessor(t *testing.T) {
	gate := assessmentgate.New(fixedAssessor{}, 1, newFakeClock(), &panicRecord{})

	got, outcome := gate.AssessmentFrom(t.Context(), address(t), nil, nil, deadline)

	if outcome != assessmentgate.Assessed || got != assessment {
		t.Fatalf("assessment = %+v, outcome %v", got, outcome)
	}
}

func TestAnAssessmentPastTheDeadlineIsGivenUp(t *testing.T) {
	clock := newFakeClock()
	assessor := newHeldAssessor()
	gate := assessmentgate.New(assessor, 1, clock, &panicRecord{})
	results := assessmentsInBackground(t, gate)

	<-assessor.started
	(<-clock.expirations)()

	if got := <-results; got.outcome != assessmentgate.AssessmentDeadline {
		t.Fatalf("outcome %v, assessment %+v", got.outcome, got.assessment)
	}
	close(assessor.released)
}

func TestAPageWaitingForAFreeSlotPastTheDeadlineMissesItsTurn(t *testing.T) {
	clock := newFakeClock()
	assessor := newHeldAssessor()
	gate := assessmentgate.New(assessor, 1, clock, &panicRecord{})
	firstResults := assessmentsInBackground(t, gate)
	<-clock.expirations
	<-assessor.started

	waitingResults := assessmentsInBackground(t, gate)
	(<-clock.expirations)()

	if got := <-waitingResults; got.outcome != assessmentgate.SlotWaitDeadline {
		t.Fatalf("outcome %v, assessment %+v", got.outcome, got.assessment)
	}
	close(assessor.released)
	if got := <-firstResults; got.outcome != assessmentgate.Assessed {
		t.Fatalf("the first page ended %v", got.outcome)
	}
}

func TestAPanickingAssessmentIsUnassessedAndReported(t *testing.T) {
	panics := &panicRecord{}
	gate := assessmentgate.New(panickingAssessor{}, 1, newFakeClock(), panics)

	_, outcome := gate.AssessmentFrom(t.Context(), address(t), nil, nil, deadline)
	_, outcomeAfterwards := gate.AssessmentFrom(t.Context(), address(t), nil, nil, deadline)

	if outcome != assessmentgate.Panicked || outcomeAfterwards != assessmentgate.Panicked ||
		len(panics.panicValues) != 2 {
		t.Fatalf("outcomes %v then %v, panics %v", outcome, outcomeAfterwards, panics.panicValues)
	}
}

func assessmentsInBackground(t *testing.T, gate *assessmentgate.Gate) <-chan result {
	t.Helper()
	results := make(chan result, 1)
	pageAddress := address(t)
	go func() {
		got, outcome := gate.AssessmentFrom(context.Background(), pageAddress, nil, nil, deadline)
		results <- result{got, outcome}
	}()
	return results
}

func address(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	pageAddress, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	return pageAddress
}
