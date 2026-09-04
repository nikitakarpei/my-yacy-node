package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	msgPendingPageVisitReturned       = "pending page visit returned for redelivery"
	msgPendingPageVisitTakenByAnother = "pending page visit dropped because another message took it"
	msgPendingPageVisitDeferred       = "pending page visit deferred"
	msgPendingPageVisitRetryScheduled = "pending page visit scheduled for another attempt"
	msgPendingPageVisitDisposedPage   = "pending page visit disposed page"
	msgPendingPageVisitCompleted      = "pending page visit completed"
)

type PendingPageVisitLog struct{}

func (PendingPageVisitLog) PendingPageVisitReturned(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	cause error,
) {
	slog.WarnContext(ctx, msgPendingPageVisitReturned,
		slog.String("order", pendingPageVisit.OrderID),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.Any("error", cause),
	)
}

func (PendingPageVisitLog) PendingPageVisitDroppedAsTakenByAnother(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	slog.DebugContext(
		ctx,
		msgPendingPageVisitTakenByAnother,
		slog.String(
			"order",
			pendingPageVisit.OrderID,
		),
		slog.String("url", pendingPageVisit.URL.String()),
	)
}

func (PendingPageVisitLog) PendingPageVisitDeferred(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	pauseFor time.Duration,
) {
	slog.DebugContext(
		ctx,
		msgPendingPageVisitDeferred,
		slog.String(
			"order",
			pendingPageVisit.OrderID,
		),
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
		slog.String(
			"order",
			pendingPageVisit.OrderID,
		),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.Duration("pauseFor", pauseFor),
	)
}

func (PendingPageVisitLog) PendingPageVisitDisposedPage(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason disposal.Reason,
) {
	slog.WarnContext(
		ctx,
		msgPendingPageVisitDisposedPage,
		slog.String(
			"order",
			pendingPageVisit.OrderID,
		),
		slog.String("url", pendingPageVisit.URL.String()),
		slog.String("reason", string(reason)),
	)
}

func (PendingPageVisitLog) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	slog.DebugContext(
		ctx,
		msgPendingPageVisitCompleted,
		slog.String(
			"order",
			pendingPageVisit.OrderID,
		),
		slog.String("url", pendingPageVisit.URL.String()),
	)
}
