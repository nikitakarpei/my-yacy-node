// Package streamopening opens, in the background, what a service reads from a stream that
// another service owns, so slower start-up steps of the service and a later start of the
// owner do not delay its consumer. While the broker or the stream is unavailable, it
// reports that and tries again after a pause that grows with each try, so the rest of the
// service keeps working meanwhile. It gives up only when the service stops.
package streamopening

import (
	"context"
	"time"
)

const (
	firstRetryPause   = time.Second
	maximumRetryPause = 30 * time.Second
)

type Clock interface {
	Sleep(ctx context.Context, pause time.Duration) error
}

type StreamOpeningObserver interface {
	StreamUnavailable(ctx context.Context, retryPause time.Duration, cause error)
	StreamOpened(ctx context.Context)
}

type Source interface {
	Close()
}

type Opening[S Source] struct {
	closed        chan struct{}
	openedSources chan S
}

func Start[S Source](
	ctx context.Context,
	open func(ctx context.Context) (S, error),
	clock Clock,
	observer StreamOpeningObserver,
) *Opening[S] {
	opening := &Opening[S]{
		closed:        make(chan struct{}),
		openedSources: make(chan S, 1),
	}
	go opening.tryUntilOpened(ctx, open, clock, observer)
	return opening
}

func (opening *Opening[S]) tryUntilOpened(
	ctx context.Context,
	open func(ctx context.Context) (S, error),
	clock Clock,
	observer StreamOpeningObserver,
) {
	defer close(opening.openedSources)
	tryCtx, stopTrying := context.WithCancel(ctx)
	defer stopTrying()
	go func() {
		select {
		case <-opening.closed:
			stopTrying()
		case <-tryCtx.Done():
		}
	}()
	retryPause := firstRetryPause
	for {
		source, err := open(tryCtx)
		if err == nil {
			observer.StreamOpened(tryCtx)
			opening.openedSources <- source
			return
		}
		observer.StreamUnavailable(tryCtx, retryPause, err)
		if clock.Sleep(tryCtx, retryPause) != nil {
			return
		}
		retryPause = min(2*retryPause, maximumRetryPause)
	}
}

func (opening *Opening[S]) Opened(ctx context.Context) (S, bool) {
	select {
	case source, opened := <-opening.openedSources:
		return source, opened
	case <-ctx.Done():
		var none S
		return none, false
	}
}

func (opening *Opening[S]) Close() {
	close(opening.closed)
	for unclaimedSource := range opening.openedSources {
		unclaimedSource.Close()
	}
}
