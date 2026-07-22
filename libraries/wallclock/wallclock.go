// Package wallclock reads the operating system clock and sleeps against it.
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

func cancellation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}
	return nil
}
