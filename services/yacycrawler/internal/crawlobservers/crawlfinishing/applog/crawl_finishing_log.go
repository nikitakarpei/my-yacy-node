// Package applog reports to the service log the crawls the crawl finishing could not
// read or finish.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	msgUnfinishedCrawlsNotRead = "the unfinished crawls could not be read to finish the empty ones"
	msgCrawlNotFinished        = "a crawl with no pending page visit could not be finished"
)

type CrawlFinishingLog struct{}

func (CrawlFinishingLog) UnfinishedCrawlsNotRead(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgUnfinishedCrawlsNotRead, slog.Any("error", err))
}

func (CrawlFinishingLog) CrawlNotFinished(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	err error,
) {
	slog.WarnContext(ctx, msgCrawlNotFinished,
		slog.String("crawl", crawlID.String()),
		slog.Any("error", err),
	)
}
