package vault_test

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var testEpoch = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

type fakeClock struct {
	guard  sync.Mutex
	now    time.Time
	sleeps chan time.Duration
	wakes  chan struct{}
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now, sleeps: make(chan time.Duration), wakes: make(chan struct{})}
}

func (c *fakeClock) Now() time.Time {
	c.guard.Lock()
	defer c.guard.Unlock()

	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case c.sleeps <- d:
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
	select {
	case <-c.wakes:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
}

func (c *fakeClock) advanceBy(d time.Duration) {
	c.guard.Lock()
	defer c.guard.Unlock()

	c.now = c.now.Add(d)
}

func (c *fakeClock) sleepAfterPass() time.Duration {
	return <-c.sleeps
}

func (c *fakeClock) wakeForPass() {
	c.wakes <- struct{}{}
}
