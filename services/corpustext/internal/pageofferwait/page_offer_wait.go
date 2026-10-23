// Package pageofferwait opens the page offers of the scrape service in the background, so
// slower start-up steps of the service do not delay its consumer. While the broker
// or its stream is unavailable, it reports that and tries again after a pause that grows
// with each try, so the rest of the service keeps working meanwhile. It gives up only
// when the service stops.
package pageofferwait

import (
	"context"
	"time"
)

const (
	firstRetryPause   = time.Second
	retryPauseMaximum = 30 * time.Second
)

type Clock interface {
	Sleep(ctx context.Context, pause time.Duration) error
}

type Observer interface {
	PageOffersUnavailable(ctx context.Context, retryPause time.Duration, cause error)
	PageOffersOpened(ctx context.Context)
}

type PageOffers interface {
	Close()
}

type Opening[Offers PageOffers] struct {
	closed       chan struct{}
	openedOffers chan Offers
}

func Start[Offers PageOffers](
	ctx context.Context,
	open func(ctx context.Context) (Offers, error),
	clock Clock,
	observer Observer,
) *Opening[Offers] {
	opening := &Opening[Offers]{
		closed:       make(chan struct{}),
		openedOffers: make(chan Offers, 1),
	}
	go func() {
		defer close(opening.openedOffers)
		tryCtx, stopTrying := context.WithCancel(ctx)
		defer stopTrying()
		go func() {
			select {
			case <-opening.closed:
				stopTrying()
			case <-tryCtx.Done():
			}
		}()
		if offers, opened := openedRetrying(tryCtx, open, clock, observer); opened {
			opening.openedOffers <- offers
		}
	}()
	return opening
}

func openedRetrying[Offers PageOffers](
	ctx context.Context,
	open func(ctx context.Context) (Offers, error),
	clock Clock,
	observer Observer,
) (Offers, bool) {
	retryPause := firstRetryPause
	for {
		offers, err := open(ctx)
		if err == nil {
			observer.PageOffersOpened(ctx)
			return offers, true
		}
		observer.PageOffersUnavailable(ctx, retryPause, err)
		if clock.Sleep(ctx, retryPause) != nil {
			var none Offers
			return none, false
		}
		retryPause = min(2*retryPause, retryPauseMaximum)
	}
}

func (opening *Opening[Offers]) Opened(ctx context.Context) (Offers, bool) {
	select {
	case offers, opened := <-opening.openedOffers:
		return offers, opened
	case <-ctx.Done():
		var none Offers
		return none, false
	}
}

func (opening *Opening[Offers]) Close() {
	close(opening.closed)
	for unclaimedOffers := range opening.openedOffers {
		unclaimedOffers.Close()
	}
}
