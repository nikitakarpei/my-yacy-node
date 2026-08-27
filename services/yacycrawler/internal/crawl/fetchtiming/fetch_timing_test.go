package fetchtiming_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchtiming"
)

type recordingObserver struct {
	elapsed []time.Duration
}

func (o *recordingObserver) FetchCompleted(elapsed time.Duration) {
	o.elapsed = append(o.elapsed, elapsed)
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

type steppingFetch struct {
	outcome pagefetch.FetchOutcome
	err     error
}

func (f *steppingFetch) Fetch(
	context.Context,
	canonicalurl.CanonicalURL,
	pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return f.outcome, f.err
}

func TestFetchReportsTheDurationTheFetchTook(t *testing.T) {
	observer := &recordingObserver{}
	clock := &steppingClock{now: time.Unix(0, 0), step: 250 * time.Millisecond}
	fetched := pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded}

	outcome, err := fetchtiming.New(observer, clock, &steppingFetch{outcome: fetched}).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://host/a"),
			pagefetch.PageVersion{},
		)
	if err != nil {
		t.Fatalf("fetch err = %v", err)
	}
	if outcome.Status != pagefetch.FetchSucceeded {
		t.Fatalf("outcome status = %v", outcome.Status)
	}
	if len(observer.elapsed) != 1 || observer.elapsed[0] != 250*time.Millisecond {
		t.Fatalf("observed elapsed = %v", observer.elapsed)
	}
}

func TestFetchReportsNothingWhenTheFetchFails(t *testing.T) {
	observer := &recordingObserver{}
	clock := &steppingClock{now: time.Unix(0, 0), step: time.Second}
	failure := errors.New("origin down")

	_, err := fetchtiming.New(observer, clock, &steppingFetch{err: failure}).
		Fetch(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, "http://host/a"),
			pagefetch.PageVersion{},
		)
	if !errors.Is(err, failure) {
		t.Fatalf("fetch err = %v", err)
	}
	if len(observer.elapsed) != 0 {
		t.Fatalf("observed elapsed = %v", observer.elapsed)
	}
}
