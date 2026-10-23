package pageofferwait_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageofferwait"
)

type firingClock struct {
	mu     sync.Mutex
	pauses []time.Duration
}

func (c *firingClock) Sleep(_ context.Context, pause time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pauses = append(c.pauses, pause)
	return nil
}

func (c *firingClock) seenPauses() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.pauses)
}

type stoppedClock struct{}

func (stoppedClock) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return fmt.Errorf("sleep: %w", ctx.Err())
}

type recordingObserver struct {
	mu                sync.Mutex
	unavailableCauses []error
	openings          int
}

func (o *recordingObserver) PageOffersUnavailable(
	_ context.Context,
	_ time.Duration,
	cause error,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.unavailableCauses = append(o.unavailableCauses, cause)
}

func (o *recordingObserver) PageOffersOpened(_ context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.openings++
}

func (o *recordingObserver) counts() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.unavailableCauses), o.openings
}

type closablePageOffers struct {
	closings *int
}

func (offers closablePageOffers) Close() {
	*offers.closings++
}

func openingAfter(failures int, closings *int) func(context.Context) (closablePageOffers, error) {
	tries := 0
	return func(context.Context) (closablePageOffers, error) {
		tries++
		if tries <= failures {
			return closablePageOffers{}, errors.New("broker unreachable")
		}
		return closablePageOffers{closings: closings}, nil
	}
}

func TestOpenedYieldsThePageOffersAtTheFirstTry(t *testing.T) {
	clock := &firingClock{}
	observer := &recordingObserver{}
	closings := 0

	opening := pageofferwait.Start(
		context.Background(),
		openingAfter(0, &closings),
		clock,
		observer,
	)
	defer opening.Close()
	offers, opened := opening.Opened(context.Background())

	if !opened || offers.closings != &closings {
		t.Fatalf("offers = %+v, opened = %v", offers, opened)
	}
	unavailable, openings := observer.counts()
	if len(clock.seenPauses()) != 0 || unavailable != 0 || openings != 1 {
		t.Errorf("pauses = %v, unavailable = %d, openings = %d",
			clock.seenPauses(), unavailable, openings)
	}
}

func TestOpenedTriesAgainAfterAGrowingPause(t *testing.T) {
	clock := &firingClock{}
	observer := &recordingObserver{}
	closings := 0

	opening := pageofferwait.Start(
		context.Background(),
		openingAfter(7, &closings),
		clock,
		observer,
	)
	defer opening.Close()
	if _, opened := opening.Opened(context.Background()); !opened {
		t.Fatal("page offers not opened")
	}

	wantPauses := []time.Duration{
		time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
		16 * time.Second, 30 * time.Second, 30 * time.Second,
	}
	if !slices.Equal(clock.seenPauses(), wantPauses) {
		t.Errorf("pauses = %v, want %v", clock.seenPauses(), wantPauses)
	}
	if unavailable, openings := observer.counts(); unavailable != 7 || openings != 1 {
		t.Errorf("unavailable = %d, openings = %d", unavailable, openings)
	}
}

func TestOpenedEndsWhenItsWaitIsCancelled(t *testing.T) {
	opening := pageofferwait.Start(
		context.Background(),
		func(context.Context) (closablePageOffers, error) {
			return closablePageOffers{}, errors.New("broker unreachable")
		},
		stoppedClock{},
		&recordingObserver{},
	)
	defer opening.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, opened := opening.Opened(ctx); opened {
		t.Error("page offers opened after the wait was cancelled")
	}
}

func TestCloseStopsTheTriesAndClosesUnclaimedPageOffers(t *testing.T) {
	closings := 0
	opening := pageofferwait.Start(
		context.Background(), openingAfter(0, &closings), &firingClock{}, &recordingObserver{},
	)

	opening.Close()

	if closings != 1 {
		t.Errorf("closings = %d, want the unclaimed page offers closed once", closings)
	}
	if _, opened := opening.Opened(context.Background()); opened {
		t.Error("page offers opened after close")
	}
}

func TestCloseStopsATryThatWaitsForTheBroker(t *testing.T) {
	observer := &recordingObserver{}
	opening := pageofferwait.Start(
		context.Background(),
		func(context.Context) (closablePageOffers, error) {
			return closablePageOffers{}, errors.New("broker unreachable")
		},
		stoppedClock{},
		observer,
	)

	opening.Close()

	if _, openings := observer.counts(); openings != 0 {
		t.Errorf("openings = %d", openings)
	}
}
