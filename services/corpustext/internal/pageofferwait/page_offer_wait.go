// Package pageofferwait opens the page offers of the scrape service. While the broker or
// its stream is unavailable, it reports that and tries again after a pause that grows
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
	After(pause time.Duration) <-chan time.Time
}

type Observer interface {
	PageOffersUnavailable(ctx context.Context, retryPause time.Duration, cause error)
	PageOffersOpened(ctx context.Context)
}

func Opened[PageOffers any](
	ctx context.Context,
	open func(ctx context.Context) (PageOffers, error),
	clock Clock,
	observer Observer,
) (PageOffers, bool) {
	retryPause := firstRetryPause
	for {
		pageOffers, err := open(ctx)
		if err == nil {
			observer.PageOffersOpened(ctx)
			return pageOffers, true
		}
		observer.PageOffersUnavailable(ctx, retryPause, err)
		select {
		case <-clock.After(retryPause):
		case <-ctx.Done():
			var none PageOffers
			return none, false
		}
		retryPause = min(2*retryPause, retryPauseMaximum)
	}
}
