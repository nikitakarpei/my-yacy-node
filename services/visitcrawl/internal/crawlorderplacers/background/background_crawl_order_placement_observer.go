package background

import "context"

type BackgroundCrawlOrderPlacementObserver interface {
	CrawlOrderPlacementAccepted(ctx context.Context, orderID string)
	CrawlOrderPlacementRefused(ctx context.Context, orderID string)
}

type BackgroundCrawlOrderPlacementObservers []BackgroundCrawlOrderPlacementObserver

func (observers BackgroundCrawlOrderPlacementObservers) CrawlOrderPlacementAccepted(
	ctx context.Context,
	orderID string,
) {
	for _, observer := range observers {
		observer.CrawlOrderPlacementAccepted(ctx, orderID)
	}
}

func (observers BackgroundCrawlOrderPlacementObservers) CrawlOrderPlacementRefused(
	ctx context.Context,
	orderID string,
) {
	for _, observer := range observers {
		observer.CrawlOrderPlacementRefused(ctx, orderID)
	}
}
