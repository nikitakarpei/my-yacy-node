package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	msgPendingPageVisitReturned       = "pending page visit returned for redelivery"
	msgPendingPageVisitDropped        = "pending page visit dropped"
	msgPendingPageVisitDeferred       = "pending page visit deferred"
	msgPendingPageVisitRetryScheduled = "pending page visit scheduled for another attempt"
	msgPendingPageVisitCompleted      = "pending page visit completed"
)

type PendingPageVisitLog struct{}

func (PendingPageVisitLog) PendingPageVisitReturned(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	cause error,
) {
	slog.WarnContext(ctx, msgPendingPageVisitReturned,
		slog.String("crawl", pendingPageVisit.CrawlID.String()),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.Any("error", cause),
	)
}

func (PendingPageVisitLog) PendingPageVisitDropped(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason pagevisitdrop.Reason,
) {
	slog.Log(ctx, levelOfDrop(reason), msgPendingPageVisitDropped,
		slog.String("crawl", pendingPageVisit.CrawlID.String()),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.String("reason", string(reason)),
	)
}

func levelOfDrop(reason pagevisitdrop.Reason) slog.Level {
	switch reason {
	case pagevisitdrop.DeferralsExhausted, pagevisitdrop.RetriesExhausted:
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}

func (PendingPageVisitLog) PendingPageVisitDeferred(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	pauseFor time.Duration,
) {
	slog.DebugContext(
		ctx,
		msgPendingPageVisitDeferred,
		slog.String("crawl", pendingPageVisit.CrawlID.String()),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.Duration("pauseFor", pauseFor),
	)
}

func (PendingPageVisitLog) PendingPageVisitRetryScheduled(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	pauseFor time.Duration,
) {
	slog.DebugContext(
		ctx,
		msgPendingPageVisitRetryScheduled,
		slog.String("crawl", pendingPageVisit.CrawlID.String()),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.Duration("pauseFor", pauseFor),
	)
}

func (PendingPageVisitLog) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	result pagevisitresult.Result,
) {
	slog.Log(ctx, levelOfResult(result), msgPendingPageVisitCompleted,
		slog.String("crawl", pendingPageVisit.CrawlID.String()),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.String("result", string(result)),
	)
}

func levelOfResult(result pagevisitresult.Result) slog.Level {
	switch result {
	case pagevisitresult.Published,
		pagevisitresult.Redirected,
		pagevisitresult.NotModified,
		pagevisitresult.AccessRefused:
		return slog.LevelDebug
	default:
		return slog.LevelWarn
	}
}
