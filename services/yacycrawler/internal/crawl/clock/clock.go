// Package clock reads the current time and waits against it.
package clock

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, duration time.Duration) error
}
