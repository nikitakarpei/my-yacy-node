package assessmentgate_test

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgate"
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
	mutex       sync.Mutex
	now         time.Time
	expirations chan func()
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: now, expirations: make(chan func(), 4)}
}

func (c *fakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *fakeClock) After(_ time.Duration, expire func()) func() {
	c.expirations <- expire
	return func() {}
}

func (c *fakeClock) passTo(elapsed time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = now.Add(elapsed)
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

type slowAssessor struct {
	clock    *fakeClock
	duration time.Duration
}

func (a slowAssessor) AssessmentFrom(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) spamassessment.Assessment {
	a.clock.passTo(a.duration)
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

type observerRecord struct {
	mutex               sync.Mutex
	slotWaits           []time.Duration
	assessmentDurations []time.Duration
	pageSizes           []int
	panicValues         []any
}

func (r *observerRecord) SlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.slotWaits = append(r.slotWaits, slotWait)
}

func (r *observerRecord) AssessmentFinished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	pageSize int,
	assessmentDuration time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pageSizes = append(r.pageSizes, pageSize)
	r.assessmentDurations = append(r.assessmentDurations, assessmentDuration)
}

func (r *observerRecord) AssessmentPanicked(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	panicValue any,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.panicValues = append(r.panicValues, panicValue)
}

func TestAnAssessmentWithinTheDeadlineIsTheAssessmentOfThePageAssessor(t *testing.T) {
	gatedAssessor := assessmentgate.New(
		fixedAssessor{},
		1,
		newFakeClock(),
		assessmentgate.Observers{&observerRecord{}},
	)

	got, outcome := gatedAssessor.Assess(t.Context(), address(t), nil, nil, deadline)

	if outcome != assessmentgate.Assessed || got != assessment {
		t.Fatalf("assessment = %+v, outcome %v", got, outcome)
	}
}

func TestAnAssessmentPastTheDeadlineIsGivenUp(t *testing.T) {
	clock := newFakeClock()
	assessor := newHeldAssessor()
	gatedAssessor := assessmentgate.New(
		assessor,
		1,
		clock,
		assessmentgate.Observers{&observerRecord{}},
	)
	results := assessmentsInBackground(t, gatedAssessor)

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
	gatedAssessor := assessmentgate.New(
		assessor,
		1,
		clock,
		assessmentgate.Observers{&observerRecord{}},
	)
	firstResults := assessmentsInBackground(t, gatedAssessor)
	<-clock.expirations
	<-assessor.started

	waitingResults := assessmentsInBackground(t, gatedAssessor)
	(<-clock.expirations)()

	if got := <-waitingResults; got.outcome != assessmentgate.SlotWaitDeadline {
		t.Fatalf("outcome %v, assessment %+v", got.outcome, got.assessment)
	}
	close(assessor.released)
	if got := <-firstResults; got.outcome != assessmentgate.Assessed {
		t.Fatalf("the first page ended %v", got.outcome)
	}
}

func TestTheTimeAndPageSizeOfAnAssessmentAreReported(t *testing.T) {
	clock := newFakeClock()
	observers := &observerRecord{}
	assessor := slowAssessor{clock: clock, duration: 30 * time.Millisecond}
	gatedAssessor := assessmentgate.New(assessor, 1, clock, assessmentgate.Observers{observers})

	gatedAssessor.Assess(t.Context(), address(t), []byte("<p>page</p>"), nil, deadline)

	if !slices.Equal(observers.assessmentDurations, []time.Duration{30 * time.Millisecond}) ||
		!slices.Equal(observers.pageSizes, []int{len("<p>page</p>")}) ||
		!slices.Equal(observers.slotWaits, []time.Duration{0}) {
		t.Fatalf("assessments took %v of pages sized %v, slot waits %v",
			observers.assessmentDurations, observers.pageSizes, observers.slotWaits)
	}
}

func TestTheSlotWaitOfAPageThatMissedItsTurnIsReported(t *testing.T) {
	clock := newFakeClock()
	observers := &observerRecord{}
	assessor := newHeldAssessor()
	gatedAssessor := assessmentgate.New(assessor, 1, clock, assessmentgate.Observers{observers})
	firstResults := assessmentsInBackground(t, gatedAssessor)
	<-clock.expirations
	<-assessor.started

	waitingResults := assessmentsInBackground(t, gatedAssessor)
	expireWaiting := <-clock.expirations
	clock.passTo(time.Second)
	expireWaiting()
	<-waitingResults
	close(assessor.released)
	<-firstResults

	if !slices.Equal(observers.slotWaits, []time.Duration{0, time.Second}) {
		t.Fatalf("slot waits %v", observers.slotWaits)
	}
}

func TestAPanickingAssessmentIsUnassessedAndReported(t *testing.T) {
	panics := &observerRecord{}
	gatedAssessor := assessmentgate.New(
		panickingAssessor{},
		1,
		newFakeClock(),
		assessmentgate.Observers{panics},
	)

	_, outcome := gatedAssessor.Assess(t.Context(), address(t), nil, nil, deadline)
	_, outcomeAfterwards := gatedAssessor.Assess(t.Context(), address(t), nil, nil, deadline)

	if outcome != assessmentgate.Panicked || outcomeAfterwards != assessmentgate.Panicked ||
		len(panics.panicValues) != 2 {
		t.Fatalf("outcomes %v then %v, panics %v", outcome, outcomeAfterwards, panics.panicValues)
	}
}

func assessmentsInBackground(t *testing.T, gatedAssessor *assessmentgate.Assessor) <-chan result {
	t.Helper()
	results := make(chan result, 1)
	pageAddress := address(t)
	go func() {
		got, outcome := gatedAssessor.Assess(context.Background(), pageAddress, nil, nil, deadline)
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
