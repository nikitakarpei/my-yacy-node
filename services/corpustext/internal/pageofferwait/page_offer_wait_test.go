package pageofferwait_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageofferwait"
)

type firingClock struct {
	pauses []time.Duration
}

func (c *firingClock) After(pause time.Duration) <-chan time.Time {
	c.pauses = append(c.pauses, pause)
	fired := make(chan time.Time, 1)
	fired <- time.Time{}
	return fired
}

type stoppedClock struct{}

func (stoppedClock) After(time.Duration) <-chan time.Time {
	return make(chan time.Time)
}

type recordingObserver struct {
	unavailableCauses []error
	openings          int
}

func (o *recordingObserver) PageOffersUnavailable(
	_ context.Context,
	_ time.Duration,
	cause error,
) {
	o.unavailableCauses = append(o.unavailableCauses, cause)
}

func (o *recordingObserver) PageOffersOpened(_ context.Context) {
	o.openings++
}

func openingAfter(failures int) func(context.Context) (string, error) {
	tries := 0
	return func(context.Context) (string, error) {
		tries++
		if tries <= failures {
			return "", errors.New("broker unreachable")
		}
		return "page offers", nil
	}
}

func TestOpenedYieldsThePageOffersAtTheFirstTry(t *testing.T) {
	clock := &firingClock{}
	observer := &recordingObserver{}

	pageOffers, opened := pageofferwait.Opened(
		context.Background(), openingAfter(0), clock, observer,
	)
	if !opened {
		t.Fatal("page offers not opened")
	}
	if pageOffers != "page offers" || len(clock.pauses) != 0 || observer.openings != 1 {
		t.Errorf("page offers = %q, pauses = %v, openings = %d",
			pageOffers, clock.pauses, observer.openings)
	}
}

func TestOpenedTriesAgainAfterAGrowingPause(t *testing.T) {
	clock := &firingClock{}
	observer := &recordingObserver{}

	pageOffers, opened := pageofferwait.Opened(
		context.Background(), openingAfter(7), clock, observer,
	)
	if !opened {
		t.Fatal("page offers not opened")
	}
	if pageOffers != "page offers" {
		t.Errorf("page offers = %q", pageOffers)
	}
	wantPauses := []time.Duration{
		time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
		16 * time.Second, 30 * time.Second, 30 * time.Second,
	}
	if !slices.Equal(clock.pauses, wantPauses) {
		t.Errorf("pauses = %v, want %v", clock.pauses, wantPauses)
	}
	if len(observer.unavailableCauses) != 7 || observer.openings != 1 {
		t.Errorf("unavailable = %d, openings = %d",
			len(observer.unavailableCauses), observer.openings)
	}
}

func TestOpenedEndsWhenTheWaitIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	observer := &recordingObserver{}
	open := func(context.Context) (string, error) {
		cancel()
		return "", errors.New("broker unreachable")
	}

	if _, opened := pageofferwait.Opened(ctx, open, stoppedClock{}, observer); opened {
		t.Error("page offers opened after the wait was cancelled")
	}
	if observer.openings != 0 {
		t.Errorf("openings = %d", observer.openings)
	}
}
