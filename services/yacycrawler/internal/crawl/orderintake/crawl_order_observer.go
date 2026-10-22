package orderintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type CrawlOrderObserver interface {
	CrawlOrderReturned(ctx context.Context, orderID string, cause error)
	CrawlOrderRejected(ctx context.Context, messageIdentity string, orderID string, cause error)
	CrawlOrderIgnored(ctx context.Context, orderID string)
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

func (observers CrawlOrderObservers) CrawlOrderRejected(
	ctx context.Context,
	messageIdentity string,
	orderID string,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawlOrderRejected(ctx, messageIdentity, orderID, cause)
	}
}

func (observers CrawlOrderObservers) CrawlOrderIgnored(ctx context.Context, orderID string) {
	for _, observer := range observers {
		observer.CrawlOrderIgnored(ctx, orderID)
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
