package pagevisitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type PendingPageVisitObserver interface {
	PendingPageVisitReturned(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		cause error,
	)
	PendingPageVisitDroppedAsTakenByAnother(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
	)
	PendingPageVisitDeferred(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		pauseFor time.Duration,
	)
	PendingPageVisitRetryScheduled(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		pauseFor time.Duration,
	)
	PendingPageVisitDisposedPage(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		reason disposal.Reason,
	)
	PendingPageVisitCompleted(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
	)
}

type PendingPageVisitObservers []PendingPageVisitObserver

func (observers PendingPageVisitObservers) PendingPageVisitReturned(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	cause error,
) {
	for _, observer := range observers {
		observer.PendingPageVisitReturned(ctx, pendingPageVisit, cause)
	}
}

func (observers PendingPageVisitObservers) PendingPageVisitDroppedAsTakenByAnother(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	for _, observer := range observers {
		observer.PendingPageVisitDroppedAsTakenByAnother(ctx, pendingPageVisit)
	}
}

func (observers PendingPageVisitObservers) PendingPageVisitDeferred(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	pauseFor time.Duration,
) {
	for _, observer := range observers {
		observer.PendingPageVisitDeferred(ctx, pendingPageVisit, pauseFor)
	}
}

func (observers PendingPageVisitObservers) PendingPageVisitRetryScheduled(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	pauseFor time.Duration,
) {
	for _, observer := range observers {
		observer.PendingPageVisitRetryScheduled(ctx, pendingPageVisit, pauseFor)
	}
}

func (observers PendingPageVisitObservers) PendingPageVisitDisposedPage(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason disposal.Reason,
) {
	for _, observer := range observers {
		observer.PendingPageVisitDisposedPage(ctx, pendingPageVisit, reason)
	}
}

func (observers PendingPageVisitObservers) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	for _, observer := range observers {
		observer.PendingPageVisitCompleted(ctx, pendingPageVisit)
	}
}
