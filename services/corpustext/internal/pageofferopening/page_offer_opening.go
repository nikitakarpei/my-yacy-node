// Package pageofferopening opens the page offers of the scrape service in the background, so
// slower start-up steps of the service do not delay its consumer. While the broker
// or its stream is unavailable, it reports that and tries again after a pause that grows
// with each try, so the rest of the service keeps working meanwhile. It gives up only
// when the service stops.
package pageofferopening

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

type PageOfferOpeningObserver interface {
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
	observer PageOfferOpeningObserver,
) *Opening[Offers] {
	opening := &Opening[Offers]{
		closed:       make(chan struct{}),
		openedOffers: make(chan Offers, 1),
	}
	go opening.tryUntilOpened(ctx, open, clock, observer)
	return opening
}

func (opening *Opening[Offers]) tryUntilOpened(
	ctx context.Context,
	open func(ctx context.Context) (Offers, error),
	clock Clock,
	observer PageOfferOpeningObserver,
) {
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
	retryPause := firstRetryPause
	for {
		offers, err := open(tryCtx)
		if err == nil {
			observer.PageOffersOpened(tryCtx)
			opening.openedOffers <- offers
			return
		}
		observer.PageOffersUnavailable(tryCtx, retryPause, err)
		if clock.Sleep(tryCtx, retryPause) != nil {
			return
		}
		retryPause = min(2*retryPause, maximumRetryPause)
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
