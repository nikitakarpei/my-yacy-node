package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	msgCrawlOrderAccepted = "crawl order accepted"
	msgCrawlOrderReturned = "crawl order returned for redelivery"
	msgCrawlOrderRejected = "crawl order rejected, nothing kept"
)

type CrawlOrderLog struct{}

func (CrawlOrderLog) CrawlOrderReturned(ctx context.Context, orderID string, cause error) {
	slog.WarnContext(ctx, msgCrawlOrderReturned,
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
}

func (CrawlOrderLog) CrawlOrderRejected(
	ctx context.Context,
	messageIdentity string,
	orderID string,
	cause error,
) {
	slog.ErrorContext(ctx, msgCrawlOrderRejected,
		slog.String("message", messageIdentity),
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
}

func (CrawlOrderLog) CrawlOrderAccepted(
	ctx context.Context,
	orderID string,
	crawlID crawlid.CrawlID,
	seedCount int,
) {
	slog.DebugContext(ctx, msgCrawlOrderAccepted,
		slog.String("order", orderID),
		slog.String("crawl", crawlID.String()),
		slog.Int("seeds", seedCount),
	)
}
