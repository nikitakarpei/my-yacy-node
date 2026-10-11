package observedfetch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/observedfetch"
)

const fetchStep = 250 * time.Millisecond

type fakeFetch struct {
	outcome pagefetch.FetchOutcome
	err     error
}

func (f fakeFetch) Fetch(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return f.outcome, f.err
}

type steppingClock struct {
	now  time.Time
	step time.Duration
}

func (c *steppingClock) Now() time.Time {
	current := c.now
	c.now = c.now.Add(c.step)
	return current
}

type recordingObserver struct {
	fetchDurations  []time.Duration
	fetched         int
	fetchesFailed   int
	fetchesCanceled int
}

func (o *recordingObserver) PageFetchSucceeded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetched++
}

func (o *recordingObserver) PageFetchNotModified(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchAccessRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchRejected(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchRedirected(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchRedirectTargetInvalid(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ error,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchRefusedOversizedPage(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
	_ error,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetchesFailed++
}

func (o *recordingObserver) PageFetchDeadlinePassed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
}

func (o *recordingObserver) PageFetchCanceled(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	duration time.Duration,
) {
	o.fetchDurations = append(o.fetchDurations, duration)
	o.fetchesCanceled++
}

func fetchObservedBy(
	ctx context.Context,
	t *testing.T,
	inner fakeFetch,
	observer *recordingObserver,
) pagefetch.FetchOutcome {
	t.Helper()
	return observedfetch.New(
		inner, &steppingClock{now: time.Unix(0, 0), step: fetchStep}, observer,
	).Fetch(ctx, canonicalurltest.CanonicalURLOf(t, "http://host/"), pagefetch.PageVersion{})
}

func TestAFetchIsObservedWithHowLongItTook(t *testing.T) {
	observer := &recordingObserver{}

	outcome := fetchObservedBy(t.Context(), t,
		fakeFetch{outcome: pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded}}, observer,
	)

	if outcome.Status != pagefetch.FetchSucceeded {
		t.Fatalf("status = %v, want the inner fetch's status", outcome.Status)
	}
	if observer.fetched != 1 {
		t.Fatalf("fetched pages observed = %d, want 1", observer.fetched)
	}
	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestAFetchThatEndsInAnErrorIsAFailedFetch(t *testing.T) {
	observer := &recordingObserver{}

	outcome := fetchObservedBy(t.Context(), t, fakeFetch{err: errors.New("boom")}, observer)

	if outcome.Status != pagefetch.FetchFailed {
		t.Fatalf("status = %v, want failed", outcome.Status)
	}
	if observer.fetchesFailed != 1 {
		t.Fatalf("failed fetches observed = %d, want 1", observer.fetchesFailed)
	}
	if len(observer.fetchDurations) != 1 || observer.fetchDurations[0] != fetchStep {
		t.Fatalf("want one %v fetch observed, got %v", fetchStep, observer.fetchDurations)
	}
}

func TestAFetchThatIsCanceledIsObservedAsCanceled(t *testing.T) {
	observer := &recordingObserver{}
	ctx, cancelFetch := context.WithCancel(t.Context())
	cancelFetch()

	fetchObservedBy(ctx, t, fakeFetch{err: errors.New("boom")}, observer)

	if observer.fetchesCanceled != 1 {
		t.Fatalf("canceled fetches = %d, want 1", observer.fetchesCanceled)
	}
}
