package visitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

type PendingVisitObserver interface {
	PendingVisitReturned(
		ctx context.Context,
		pendingVisit pendingvisit.PendingVisit,
		cause error,
	)
	PendingVisitDroppedBecauseClaimedElsewhere(
		ctx context.Context,
		pendingVisit pendingvisit.PendingVisit,
	)
	PendingVisitDeferred(
		ctx context.Context,
		pendingVisit pendingvisit.PendingVisit,
		pauseFor time.Duration,
	)
	PendingVisitRetryScheduled(
		ctx context.Context,
		pendingVisit pendingvisit.PendingVisit,
		pauseFor time.Duration,
	)
	PendingVisitDisposedPage(
		ctx context.Context,
		pendingVisit pendingvisit.PendingVisit,
		reason disposal.Reason,
	)
	PendingVisitCompleted(ctx context.Context, pendingVisit pendingvisit.PendingVisit)
}

type PendingVisitObservers []PendingVisitObserver

func (observers PendingVisitObservers) PendingVisitReturned(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	cause error,
) {
	for _, observer := range observers {
		observer.PendingVisitReturned(ctx, pendingVisit, cause)
	}
}

func (observers PendingVisitObservers) PendingVisitDroppedBecauseClaimedElsewhere(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
) {
	for _, observer := range observers {
		observer.PendingVisitDroppedBecauseClaimedElsewhere(ctx, pendingVisit)
	}
}

func (observers PendingVisitObservers) PendingVisitDeferred(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	pauseFor time.Duration,
) {
	for _, observer := range observers {
		observer.PendingVisitDeferred(ctx, pendingVisit, pauseFor)
	}
}

func (observers PendingVisitObservers) PendingVisitRetryScheduled(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	pauseFor time.Duration,
) {
	for _, observer := range observers {
		observer.PendingVisitRetryScheduled(ctx, pendingVisit, pauseFor)
	}
}

func (observers PendingVisitObservers) PendingVisitDisposedPage(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
	reason disposal.Reason,
) {
	for _, observer := range observers {
		observer.PendingVisitDisposedPage(ctx, pendingVisit, reason)
	}
}

func (observers PendingVisitObservers) PendingVisitCompleted(
	ctx context.Context,
	pendingVisit pendingvisit.PendingVisit,
) {
	for _, observer := range observers {
		observer.PendingVisitCompleted(ctx, pendingVisit)
	}
}
