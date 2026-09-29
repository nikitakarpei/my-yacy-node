// Package readingcancel cancels reading an upstream response that passes its
// deadline or waits too long for its next bytes.
package readingcancel

import (
	"context"
	"errors"
	"time"
)

var ErrDeadlinePassed = errors.New("reading deadline passed")

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

func CancelAt(
	clock Clock,
	deadline time.Time,
	cancelReading context.CancelCauseFunc,
) (stop func()) {
	return clock.After(deadline.Sub(clock.Now()), func() { cancelReading(ErrDeadlinePassed) })
}
