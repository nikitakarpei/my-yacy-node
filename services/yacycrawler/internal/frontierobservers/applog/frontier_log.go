// Package applog reports to the service log the frontier operations the crawl state storage
// refused.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	msgFrontierLeaseFailed   = "no page visit could be leased from the frontier"
	msgFrontierDrainFailed   = "the page visits of a draining queue could not be dropped"
	msgFrontierCountFailed   = "the page visits on the frontier could not be counted"
	msgFrontierEndFailed     = "an ended page visit could not be removed from the frontier"
	msgFrontierReturnFailed  = "a page visit could not be returned to the frontier"
	msgFrontierLeaseOutdated = "a page visit was ended or returned under a lease that has expired"
)

type FrontierLog struct{}

func (FrontierLog) FrontierLeaseFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierLeaseFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierCountFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierCountFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierEndFailed(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	err error,
) {
	slog.WarnContext(ctx, msgFrontierEndFailed,
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

func (FrontierLog) FrontierDrainFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierDrainFailed, slog.Any("error", err))
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
