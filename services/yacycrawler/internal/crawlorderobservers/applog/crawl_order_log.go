package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlOrderAccepted = "crawl order accepted"
	msgCrawlOrderReturned = "crawl order returned for redelivery"
)

type CrawlOrderLog struct{}

func (CrawlOrderLog) CrawlOrderReturned(ctx context.Context, orderID string, cause error) {
	slog.WarnContext(ctx, msgCrawlOrderReturned,
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
}

func (CrawlOrderLog) CrawlOrderAccepted(ctx context.Context, orderID string, seedCount int) {
	slog.DebugContext(ctx, msgCrawlOrderAccepted,
		slog.String("order", orderID),
		slog.Int("seeds", seedCount),
	)
}
