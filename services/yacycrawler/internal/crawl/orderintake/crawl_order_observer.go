package orderintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type CrawlOrderObserver interface {
	CrawlOrderReturned(ctx context.Context, orderID string, cause error)
	CrawlOrderAccepted(
		ctx context.Context,
		orderID string,
		crawlID crawlid.CrawlID,
		seedCount int,
	)
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
	crawlID crawlid.CrawlID,
	seedCount int,
) {
	for _, observer := range observers {
		observer.CrawlOrderAccepted(ctx, orderID, crawlID, seedCount)
	}
}
