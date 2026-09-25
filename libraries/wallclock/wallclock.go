// Package wallclock reads the operating system clock, sleeps against it, and
// runs a function once a timeout passed on it.
package wallclock

import (
	"context"
	"fmt"
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now()
}

func (Clock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return cancellation(ctx)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return cancellation(ctx)
	case <-timer.C:
		return nil
	}
}

func (Clock) After(timeout time.Duration, expire func()) (stop func()) {
	timer := time.AfterFunc(timeout, expire)

	return func() { timer.Stop() }
}

func cancellation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	return nil
}
