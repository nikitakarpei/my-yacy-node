package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlOrderPlacementAccepted = "crawl order placement accepted"
	msgCrawlOrderPlacementRefused  = "crawl order placement refused"
)

type BackgroundCrawlOrderPlacementLog struct{}

func (BackgroundCrawlOrderPlacementLog) CrawlOrderPlacementAccepted(
	ctx context.Context,
	orderID string,
) {
	slog.DebugContext(ctx, msgCrawlOrderPlacementAccepted, slog.String("order", orderID))
}

func (BackgroundCrawlOrderPlacementLog) CrawlOrderPlacementRefused(
	ctx context.Context,
	orderID string,
) {
	slog.WarnContext(ctx, msgCrawlOrderPlacementRefused, slog.String("order", orderID))
}
