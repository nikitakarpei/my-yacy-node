package vaultenginetest

import (
	"context"
	"fmt"
	"time"
)

type FrozenClock struct {
	At time.Time
}

func (c FrozenClock) Now() time.Time {
	return c.At
}

func (FrozenClock) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()

	return fmt.Errorf("context: %w", ctx.Err())
}
