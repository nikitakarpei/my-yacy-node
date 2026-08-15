package rwiescrow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
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

func runLoopUntilFirstRun(
	t *testing.T,
	expiry *countingExpiry,
	observer rwiescrow.ExpiryObserver,
) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		rwiescrow.RunExpiryLoop(ctx, expiry, observer, rwiescrow.ExpiryConfig{
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

type drainingExpiry struct {
	remaining int
	drained   chan struct{}
}

func (d *drainingExpiry) Expire(_ context.Context, _ time.Duration, limit int) (int, error) {
	expired := min(d.remaining, limit)
	d.remaining -= expired
	if d.remaining == 0 {
		close(d.drained)
	}

	return expired, nil
}

func TestExpiryLoopDrainsEveryBatchBeforeItsNextTick(t *testing.T) {
	observer := &countingExpiries{}
	expiry := &drainingExpiry{remaining: 25, drained: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		rwiescrow.RunExpiryLoop(ctx, expiry, observer, rwiescrow.ExpiryConfig{
			HoldFor:  holdFor,
			Interval: time.Hour,
			Batch:    10,
		})
		close(done)
	}()

	<-expiry.drained
	cancel()
	<-done

	if observer.expired != 25 {
		t.Fatalf("observed expiries = %d, want every held posting drained", observer.expired)
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
