package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

const (
	msgPendingVisitReturned         = "pending visit returned for redelivery"
	msgPendingVisitClaimedElsewhere = "pending visit dropped because another message holds its claim"
	msgPendingVisitDeferred         = "pending visit deferred"
	msgPendingVisitRetryScheduled   = "pending visit scheduled for another attempt"
	msgPendingVisitDisposedPage     = "pending visit disposed page"
	msgPendingVisitCompleted        = "pending visit completed"
)

type PendingVisitLog struct{}

func (PendingVisitLog) PendingVisitReturned(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	cause error,
) {
	slog.WarnContext(ctx, msgPendingVisitReturned,
		slog.String("order", pendingVisit.OrderID),
		slog.String("url", pendingVisit.URL.String()),
		slog.Any("error", cause),
	)
}

func (PendingVisitLog) PendingVisitDroppedBecauseClaimedElsewhere(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
) {
	slog.DebugContext(ctx, msgPendingVisitClaimedElsewhere,
		slog.String("order", pendingVisit.OrderID), slog.String("url", pendingVisit.URL.String()))
}

func (PendingVisitLog) PendingVisitDeferred(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	pauseFor time.Duration,
) {
	slog.DebugContext(ctx, msgPendingVisitDeferred,
		slog.String("order", pendingVisit.OrderID), slog.String("url", pendingVisit.URL.String()),
		slog.Duration("pauseFor", pauseFor))
}

func (PendingVisitLog) PendingVisitRetryScheduled(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	pauseFor time.Duration,
) {
	slog.DebugContext(ctx, msgPendingVisitRetryScheduled,
		slog.String("order", pendingVisit.OrderID), slog.String("url", pendingVisit.URL.String()),
		slog.Duration("pauseFor", pauseFor))
}

func (PendingVisitLog) PendingVisitDisposedPage(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	reason disposal.Reason,
) {
	slog.WarnContext(ctx, msgPendingVisitDisposedPage,
		slog.String("order", pendingVisit.OrderID), slog.String("url", pendingVisit.URL.String()),
		slog.String("reason", string(reason)))
}

func (PendingVisitLog) PendingVisitCompleted(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
) {
	slog.DebugContext(ctx, msgPendingVisitCompleted,
		slog.String("order", pendingVisit.OrderID), slog.String("url", pendingVisit.URL.String()))
}
