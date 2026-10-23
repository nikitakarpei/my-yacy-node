// Package applog reports to the service log the page visits and links the crawl progress
// could not count.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	msgCompletedVisitNotCounted  = "a completed page visit could not be counted in the progress of its crawl"
	msgDroppedVisitNotCounted    = "a dropped page visit could not be counted in the progress of its crawl"
	msgUnadmittedLinksNotCounted = "unadmitted links could not be counted in the progress of their crawl"
)

type CrawlProgressLog struct{}

func (CrawlProgressLog) CompletedPageVisitNotCounted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	result yacycrawlcontract.PageVisitResult,
	err error,
) {
	slog.WarnContext(ctx, msgCompletedVisitNotCounted,
		slog.String("crawl", crawlID.String()),
		slog.String("result", string(result)),
		slog.Any("error", err),
	)
}

func (CrawlProgressLog) DroppedPageVisitNotCounted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	reason yacycrawlcontract.PageVisitDropReason,
	err error,
) {
	slog.WarnContext(ctx, msgDroppedVisitNotCounted,
		slog.String("crawl", crawlID.String()),
		slog.String("reason", string(reason)),
		slog.Any("error", err),
	)
}

func (CrawlProgressLog) UnadmittedLinksNotCounted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	err error,
) {
	slog.WarnContext(ctx, msgUnadmittedLinksNotCounted,
		slog.String("crawl", crawlID.String()),
		slog.Any("error", err),
	)
}
