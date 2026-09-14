package orderintake

import "context"

type CrawlOrderObserver interface {
	CrawlOrderReturned(ctx context.Context, orderID string, cause error)
	CrawlOrderAccepted(ctx context.Context, orderID string, seedCount int)
}

type CrawlOrderObservers []CrawlOrderObserver

func (observers CrawlOrderObservers) CrawlOrderReturned(
	ctx context.Context,
	orderID string,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawlOrderReturned(ctx, orderID, cause)
	}
}

func (observers CrawlOrderObservers) CrawlOrderAccepted(
	ctx context.Context,
	orderID string,
	seedCount int,
) {
	for _, observer := range observers {
		observer.CrawlOrderAccepted(ctx, orderID, seedCount)
	}
}
