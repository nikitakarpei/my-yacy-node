package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlOrderPlaced                    = "crawl order placed"
	msgCrawlOrderPlacementFailed           = "crawl order placement failed"
	msgCrawlOrderPlacementSkippedSaturated = "crawl order placement skipped: capacity saturated"
)

type CrawlOrderPlacementLog struct{}

func (CrawlOrderPlacementLog) CrawlOrderPlaced(
	ctx context.Context,
	orderID string,
) {
	slog.DebugContext(ctx, msgCrawlOrderPlaced, slog.String("order", orderID))
}

func (CrawlOrderPlacementLog) CrawlOrderPlacementFailed(
	ctx context.Context,
	orderID string,
	cause error,
) {
	slog.WarnContext(ctx, msgCrawlOrderPlacementFailed,
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
}

func (CrawlOrderPlacementLog) CrawlOrderPlacementSkippedBecauseSaturated(
	ctx context.Context,
	orderID string,
) {
	slog.WarnContext(ctx, msgCrawlOrderPlacementSkippedSaturated, slog.String("order", orderID))
}
