// Package applog reports to the service log the crawls the crawl settlement could not
// read or settle.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	msgUnfinishedCrawlsNotRead = "the unfinished crawls could not be read to settle the empty ones"
	msgCrawlNotSettled         = "a crawl with no pending page visit could not be settled"
)

type CrawlSettlementLog struct{}

func (CrawlSettlementLog) UnfinishedCrawlsNotRead(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgUnfinishedCrawlsNotRead, slog.Any("error", err))
}

func (CrawlSettlementLog) CrawlNotSettled(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	err error,
) {
	slog.WarnContext(ctx, msgCrawlNotSettled,
		slog.String("crawl", crawlID.String()),
		slog.Any("error", err),
	)
}
