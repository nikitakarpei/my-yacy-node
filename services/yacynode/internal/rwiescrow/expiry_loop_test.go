package rwiescrow

import (
	"context"
	"errors"
	"testing"
	"time"
)

type countingExpiry struct {
	runs  chan struct{}
	fails bool
}

func (c *countingExpiry) Expire(context.Context, time.Duration, int) (int, error) {
	c.runs <- struct{}{}
	if c.fails {
		return 0, errors.New("expiry unavailable")
	}

	return 1, nil
}

type countingExpiries struct {
	expired  int
	failures int
}

func (c *countingExpiries) ObserveExpired(postings int) { c.expired += postings }
func (c *countingExpiries) ObserveExpiryFailure()       { c.failures++ }

func runLoopUntilFirstRun(t *testing.T, expiry *countingExpiry, observer ExpiryObserver) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		RunExpiryLoop(ctx, expiry, observer, ExpiryConfig{
			HoldFor:  holdFor,
			Interval: time.Hour,
			Batch:    10,
		})
		close(done)
	}()

	<-expiry.runs
	cancel()
	<-done
}

func TestExpiryLoopExpiresBeforeItsFirstTick(t *testing.T) {
	observer := &countingExpiries{}
	runLoopUntilFirstRun(t, &countingExpiry{runs: make(chan struct{}, 1)}, observer)

	if observer.expired != 1 {
		t.Fatalf("observed expiries = %d, want 1", observer.expired)
	}
}

func TestExpiryLoopReportsFailure(t *testing.T) {
	observer := &countingExpiries{}
	runLoopUntilFirstRun(
		t,
		&countingExpiry{runs: make(chan struct{}, 1), fails: true},
		observer,
	)

	if observer.failures != 1 {
		t.Fatalf("observed failures = %d, want 1", observer.failures)
	}
}
