package classificationgate_test

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgate"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

var (
	now            = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	classification = pageclassification.Classification{
		pageclassification.Spam: {Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"},
	}
)

type result struct {
	classification pageclassification.Classification
	outcome        classificationgate.Outcome
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

type heldClassifier struct {
	started  chan struct{}
	released chan struct{}
}

func newHeldClassifier() heldClassifier {
	return heldClassifier{started: make(chan struct{}, 2), released: make(chan struct{})}
}

func (a heldClassifier) ClassificationOf(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) pageclassification.Classification {
	a.started <- struct{}{}
	<-a.released
	return classification
}

type fixedClassifier struct{}

func (fixedClassifier) ClassificationOf(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) pageclassification.Classification {
	return classification
}

type slowClassifier struct {
	clock    *fakeClock
	duration time.Duration
}

func (a slowClassifier) ClassificationOf(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) pageclassification.Classification {
	a.clock.passTo(a.duration)
	return classification
}

type panickingClassifier struct{}

func (panickingClassifier) ClassificationOf(
	canonicalurl.CanonicalURL,
	[]byte,
	http.Header,
) pageclassification.Classification {
	panic("broken page")
}

type observerRecord struct {
	mutex                   sync.Mutex
	slotWaits               []time.Duration
	classificationDurations []time.Duration
	pageSizes               []int
	panicValues             []any
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

func (r *observerRecord) ClassificationFinished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	pageSize int,
	classificationDuration time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pageSizes = append(r.pageSizes, pageSize)
	r.classificationDurations = append(r.classificationDurations, classificationDuration)
}

func (r *observerRecord) ClassificationPanicked(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	panicValue any,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.panicValues = append(r.panicValues, panicValue)
}

func TestAClassificationIsTheClassificationOfTheClassifier(t *testing.T) {
	classificationRunner := classificationgate.New(
		fixedClassifier{},
		1,
		newFakeClock(),
		classificationgate.Observers{&observerRecord{}},
	)

	got, outcome := classificationRunner.Classify(t.Context(), address(t), nil, nil)

	if outcome != classificationgate.Classified || !maps.Equal(got, classification) {
		t.Fatalf("classification = %+v, outcome %v", got, outcome)
	}
}

func TestAClassificationWhoseRoundTripIsCancelledIsGivenUp(t *testing.T) {
	classifier := newHeldClassifier()
	classificationRunner := classificationgate.New(
		classifier,
		1,
		newFakeClock(),
		classificationgate.Observers{&observerRecord{}},
	)
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	results := classificationsInBackground(ctx, t, classificationRunner)

	<-classifier.started
	cancelRoundTrip()

	if got := <-results; got.outcome != classificationgate.ClassificationCancelled {
		t.Fatalf("outcome %v, classification %+v", got.outcome, got.classification)
	}
	close(classifier.released)
}

func TestAPageWhoseRoundTripIsCancelledWhileWaitingForAFreeSlotMissesItsTurn(t *testing.T) {
	classifier := newHeldClassifier()
	classificationRunner := classificationgate.New(
		classifier,
		1,
		newFakeClock(),
		classificationgate.Observers{&observerRecord{}},
	)
	firstResults := classificationsInBackground(t.Context(), t, classificationRunner)
	<-classifier.started
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	cancelRoundTrip()

	waitingResults := classificationsInBackground(ctx, t, classificationRunner)

	if got := <-waitingResults; got.outcome != classificationgate.SlotWaitCancelled {
		t.Fatalf("outcome %v, classification %+v", got.outcome, got.classification)
	}
	close(classifier.released)
	if got := <-firstResults; got.outcome != classificationgate.Classified {
		t.Fatalf("the first page ended %v", got.outcome)
	}
}

func TestTheTimeAndPageSizeOfAClassificationAreReported(t *testing.T) {
	clock := newFakeClock()
	observers := &observerRecord{}
	classifier := slowClassifier{clock: clock, duration: 30 * time.Millisecond}
	classificationRunner := classificationgate.New(
		classifier,
		1,
		clock,
		classificationgate.Observers{observers},
	)

	classificationRunner.Classify(t.Context(), address(t), []byte("<p>page</p>"), nil)

	if !slices.Equal(observers.classificationDurations, []time.Duration{30 * time.Millisecond}) ||
		!slices.Equal(observers.pageSizes, []int{len("<p>page</p>")}) ||
		!slices.Equal(observers.slotWaits, []time.Duration{0}) {
		t.Fatalf("classifications took %v of pages sized %v, slot waits %v",
			observers.classificationDurations, observers.pageSizes, observers.slotWaits)
	}
}

func TestTheSlotWaitOfAPageThatMissedItsTurnIsReported(t *testing.T) {
	clock := newFakeClock()
	observers := &observerRecord{}
	classifier := newHeldClassifier()
	classificationRunner := classificationgate.New(
		classifier,
		1,
		clock,
		classificationgate.Observers{observers},
	)
	firstResults := classificationsInBackground(t.Context(), t, classificationRunner)
	<-classifier.started
	ctx, cancelRoundTrip := context.WithCancel(t.Context())
	cancelRoundTrip()
	clock.passAfterNextReading(time.Second)

	classificationRunner.Classify(ctx, address(t), nil, nil)
	close(classifier.released)
	<-firstResults

	if !slices.Equal(observers.slotWaits, []time.Duration{0, time.Second}) {
		t.Fatalf("slot waits %v", observers.slotWaits)
	}
}

func TestAPanickingClassificationIsUnclassifiedAndReported(t *testing.T) {
	panics := &observerRecord{}
	classificationRunner := classificationgate.New(
		panickingClassifier{},
		1,
		newFakeClock(),
		classificationgate.Observers{panics},
	)

	_, outcome := classificationRunner.Classify(t.Context(), address(t), nil, nil)
	_, outcomeAfterwards := classificationRunner.Classify(t.Context(), address(t), nil, nil)

	if outcome != classificationgate.Panicked || outcomeAfterwards != classificationgate.Panicked ||
		len(panics.panicValues) != 2 {
		t.Fatalf("outcomes %v then %v, panics %v", outcome, outcomeAfterwards, panics.panicValues)
	}
}

func classificationsInBackground(
	ctx context.Context,
	t *testing.T,
	classificationRunner *classificationgate.Runner,
) <-chan result {
	t.Helper()
	results := make(chan result, 1)
	pageAddress := address(t)
	go func() {
		got, outcome := classificationRunner.Classify(ctx, pageAddress, nil, nil)
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
