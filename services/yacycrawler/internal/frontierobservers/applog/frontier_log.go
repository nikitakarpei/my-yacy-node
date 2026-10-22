// Package applog reports to the service log the frontier and crawl operations the crawl
// state storage refused.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	msgFrontierLeaseFailed       = "no page visit could be leased from the frontier"
	msgFrontierCountFailed       = "the page visits on the frontier could not be counted"
	msgFrontierAcknowledgeFailed = "a settled page visit could not be removed from the frontier"
	msgFrontierReturnFailed      = "a page visit could not be returned to the frontier"
	msgFrontierCrawlNotSettled   = "a crawl that owes no page visit could not be finished"
	msgFrontierLeaseOutdated     = "a page visit was settled under a lease that has since ended"
)

type FrontierLog struct{}

func (FrontierLog) FrontierLeaseFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierLeaseFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierCountFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierCountFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierAcknowledgeFailed(ctx context.Context, identity string, err error) {
	slog.WarnContext(ctx, msgFrontierAcknowledgeFailed,
		slog.String("pageVisit", identity),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierReturnFailed(
	ctx context.Context,
	identity string,
	delay time.Duration,
	err error,
) {
	slog.WarnContext(ctx, msgFrontierReturnFailed,
		slog.String("pageVisit", identity),
		slog.Duration("delay", delay),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierCrawlNotSettled(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	err error,
) {
	slog.ErrorContext(ctx, msgFrontierCrawlNotSettled,
		slog.String("crawl", crawlID.String()),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierLeaseOutdated(ctx context.Context, identity string) {
	slog.DebugContext(ctx, msgFrontierLeaseOutdated, slog.String("pageVisit", identity))
}
