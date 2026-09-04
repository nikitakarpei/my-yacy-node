package crawlorderbroker

import "context"

type CrawlOrderPublicationObserver interface {
	CrawlOrderPublished(ctx context.Context, orderID string, subject string)
	CrawlOrderPublishingFailed(
		ctx context.Context,
		orderID string,
		subject string,
		cause error,
	)
	CrawlOrderEncodingFailed(ctx context.Context, orderID string, cause error)
}

type CrawlOrderPublicationObservers []CrawlOrderPublicationObserver

func (observers CrawlOrderPublicationObservers) CrawlOrderPublished(
	ctx context.Context,
	orderID string,
	subject string,
) {
	for _, observer := range observers {
		observer.CrawlOrderPublished(ctx, orderID, subject)
	}
}

func (observers CrawlOrderPublicationObservers) CrawlOrderPublishingFailed(
	ctx context.Context,
	orderID string,
	subject string,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawlOrderPublishingFailed(ctx, orderID, subject, cause)
	}
}

func (observers CrawlOrderPublicationObservers) CrawlOrderEncodingFailed(
	ctx context.Context,
	orderID string,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawlOrderEncodingFailed(ctx, orderID, cause)
	}
}
