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
	assessment = spamassessment.Assessment{Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"}
)

type result struct {
	assessment spamassessment.Assessment
	outcome    assessmentgate.Outcome
}

type fakeClock struct {
	mutex            sync.Mutex
	now              time.Time
	passAfterReading time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: now}
}

func (c *fakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	reading := c.now
	c.now = c.now.Add(c.passAfterReading)
	c.passAfterReading = 0
	return reading
}

func (c *fakeClock) passAfterNextReading(elapsed time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.passAfterReading = elapsed
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

func TestAnAssessmentIsTheAssessmentOfTheAssessor(t *testing.T) {
	assessmentRunner := assessmentgate.New(
		fixedAssessor{},
		1,
		newFakeClock(),
		assessmentgate.Observers{&observerRecord{}},
	)

	got, outcome := assessmentRunner.Assess(t.Context(), address(t), nil, nil)

	if outcome != assessmentgate.Assessed || got != assessment {
		t.Fatalf("assessment = %+v, outcome %v", got, outcome)
	}
}

func TestAnAssessmentWhoseRoundTripIsCancelledIsGivenUp(t *testing.T) {
	assessor := newHeldAssessor()
	assessmentRunner := assessmentgate.New(
		assessor,
		1,
		newFakeClock(),
		assessmentgate.Observers{&observerRecord{}},
	)
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	results := assessmentsInBackground(ctx, t, assessmentRunner)

	<-assessor.started
	cancelRoundTrip()

	if got := <-results; got.outcome != assessmentgate.AssessmentCancelled {
		t.Fatalf("outcome %v, assessment %+v", got.outcome, got.assessment)
	}
	close(assessor.released)
}

func TestAPageWhoseRoundTripIsCancelledWhileWaitingForAFreeSlotMissesItsTurn(t *testing.T) {
	assessor := newHeldAssessor()
	assessmentRunner := assessmentgate.New(
		assessor,
		1,
		newFakeClock(),
		assessmentgate.Observers{&observerRecord{}},
	)
	firstResults := assessmentsInBackground(t.Context(), t, assessmentRunner)
	<-assessor.started
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	cancelRoundTrip()

	waitingResults := assessmentsInBackground(ctx, t, assessmentRunner)

	if got := <-waitingResults; got.outcome != assessmentgate.SlotWaitCancelled {
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
	assessmentRunner := assessmentgate.New(assessor, 1, clock, assessmentgate.Observers{observers})

	assessmentRunner.Assess(t.Context(), address(t), []byte("<p>page</p>"), nil)

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
	assessmentRunner := assessmentgate.New(assessor, 1, clock, assessmentgate.Observers{observers})
	firstResults := assessmentsInBackground(t.Context(), t, assessmentRunner)
	<-assessor.started
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	cancelRoundTrip()
	clock.passAfterNextReading(time.Second)

	assessmentRunner.Assess(ctx, address(t), nil, nil)
	close(assessor.released)
	<-firstResults

	if !slices.Equal(observers.slotWaits, []time.Duration{0, time.Second}) {
		t.Fatalf("slot waits %v", observers.slotWaits)
	}
}

func TestAPanickingAssessmentIsUnassessedAndReported(t *testing.T) {
	panics := &observerRecord{}
	assessmentRunner := assessmentgate.New(
		panickingAssessor{},
		1,
		newFakeClock(),
		assessmentgate.Observers{panics},
	)

	_, outcome := assessmentRunner.Assess(t.Context(), address(t), nil, nil)
	_, outcomeAfterwards := assessmentRunner.Assess(t.Context(), address(t), nil, nil)

	if outcome != assessmentgate.Panicked || outcomeAfterwards != assessmentgate.Panicked ||
		len(panics.panicValues) != 2 {
		t.Fatalf("outcomes %v then %v, panics %v", outcome, outcomeAfterwards, panics.panicValues)
	}
}

func assessmentsInBackground(
	ctx context.Context,
	t *testing.T,
	assessmentRunner *assessmentgate.Runner,
) <-chan result {
	t.Helper()
	results := make(chan result, 1)
	pageAddress := address(t)
	go func() {
		got, outcome := assessmentRunner.Assess(ctx, pageAddress, nil, nil)
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
