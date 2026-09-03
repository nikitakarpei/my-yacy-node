package visitintake

import (
	"context"
)

type CrawlOrderPlacementObserver interface {
	CrawlOrderPlaced(ctx context.Context, orderID string)
	CrawlOrderPlacementFailed(
		ctx context.Context,
		orderID string,
		cause error,
	)
	CrawlOrderPlacementSkippedBecauseSaturated(ctx context.Context, orderID string)
}

type CrawlOrderPlacementObservers []CrawlOrderPlacementObserver

func (observers CrawlOrderPlacementObservers) CrawlOrderPlaced(
	ctx context.Context,
	orderID string,
) {
	for _, observer := range observers {
		observer.CrawlOrderPlaced(ctx, orderID)
	}
}

func (observers CrawlOrderPlacementObservers) CrawlOrderPlacementFailed(
	ctx context.Context,
	orderID string,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawlOrderPlacementFailed(ctx, orderID, cause)
	}
}

func (observers CrawlOrderPlacementObservers) CrawlOrderPlacementSkippedBecauseSaturated(
	ctx context.Context,
	orderID string,
) {
	for _, observer := range observers {
		observer.CrawlOrderPlacementSkippedBecauseSaturated(ctx, orderID)
	}
}
