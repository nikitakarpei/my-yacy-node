package distributionloop

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
	After(timeout time.Duration, expire func()) (stop func())
}

func sleepOn(ctx context.Context, clock Clock, pause time.Duration) {
	elapsed := make(chan struct{})
	stop := clock.After(pause, func() { close(elapsed) })
	defer stop()

	select {
	case <-ctx.Done():
	case <-elapsed:
	}
}

func waitOn(ctx context.Context, clock Clock, wake <-chan struct{}, alarm wakeAlarm) {
	elapsed := make(chan struct{})
	if alarm.isSet {
		stop := clock.After(max(alarm.at.Sub(clock.Now()), 0), func() { close(elapsed) })
		defer stop()
	}

	select {
	case <-ctx.Done():
	case <-wake:
	case <-elapsed:
	}
}
