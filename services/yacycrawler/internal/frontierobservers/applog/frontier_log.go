// Package applog reports to the service log the frontier and crawl operations the crawl
// state storage refused.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	msgFrontierLeaseFailed       = "no page visit could be leased from the frontier"
	msgFrontierDrainFailed       = "the page visits of a cancelled crawl could not be drained"
	msgFrontierCountFailed       = "the page visits on the frontier could not be counted"
	msgFrontierSettleFailed      = "a settled page visit could not be removed from the frontier"
	msgFrontierReturnFailed      = "a page visit could not be returned to the frontier"
	msgCompletedVisitNotCounted  = "a completed page visit could not be counted in the progress of its crawl"
	msgDroppedVisitNotCounted    = "a dropped page visit could not be counted in the progress of its crawl"
	msgUnadmittedLinksNotCounted = "unadmitted links could not be counted in the progress of their crawl"
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

func (FrontierLog) FrontierSettleFailed(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	err error,
) {
	slog.WarnContext(ctx, msgFrontierSettleFailed,
		slog.String("crawl", pageVisit.CrawlID.String()),
		slog.String("url", pageVisit.URL.String()),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierReturnFailed(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	delay time.Duration,
	err error,
) {
	slog.WarnContext(ctx, msgFrontierReturnFailed,
		slog.String("crawl", pageVisit.CrawlID.String()),
		slog.String("url", pageVisit.URL.String()),
		slog.Duration("delay", delay),
		slog.Any("error", err),
	)
}

func (FrontierLog) CompletedPageVisitNotCounted(
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

func (FrontierLog) DroppedPageVisitNotCounted(
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

func (FrontierLog) UnadmittedLinksNotCounted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	err error,
) {
	slog.WarnContext(ctx, msgUnadmittedLinksNotCounted,
		slog.String("crawl", crawlID.String()),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierDrainFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierDrainFailed, slog.Any("error", err))
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

func (FrontierLog) FrontierLeaseOutdated(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) {
	slog.DebugContext(ctx, msgFrontierLeaseOutdated,
		slog.String("crawl", pageVisit.CrawlID.String()),
		slog.String("url", pageVisit.URL.String()),
	)
}
