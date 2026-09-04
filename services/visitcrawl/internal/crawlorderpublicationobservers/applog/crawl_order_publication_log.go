package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlOrderPublished        = "crawl order published"
	msgCrawlOrderPublishingFailed = "crawl order publishing failed"
	msgCrawlOrderEncodingFailed   = "crawl order encoding failed"
)

type CrawlOrderPublicationLog struct{}

func (CrawlOrderPublicationLog) CrawlOrderPublished(
	ctx context.Context,
	orderID string,
	subject string,
) {
	slog.DebugContext(ctx, msgCrawlOrderPublished,
		slog.String("order", orderID),
		slog.String("subject", subject),
	)
}

func (CrawlOrderPublicationLog) CrawlOrderPublishingFailed(
	ctx context.Context,
	orderID string,
	subject string,
	cause error,
) {
	slog.WarnContext(ctx, msgCrawlOrderPublishingFailed,
		slog.String("order", orderID),
		slog.String("subject", subject),
		slog.Any("error", cause),
	)
}

func (CrawlOrderPublicationLog) CrawlOrderEncodingFailed(
	ctx context.Context,
	orderID string,
	cause error,
) {
	slog.ErrorContext(ctx, msgCrawlOrderEncodingFailed,
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
}
