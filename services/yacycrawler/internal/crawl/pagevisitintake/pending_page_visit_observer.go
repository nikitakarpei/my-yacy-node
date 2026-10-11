package pagevisitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type PendingPageVisitObserver interface {
	PendingPageVisitReturned(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		cause error,
	)
	PendingPageVisitDropped(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		reason pagevisitdrop.Reason,
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
	PendingPageVisitCompleted(
		ctx context.Context,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
		result pagevisitresult.Result,
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

func (observers PendingPageVisitObservers) PendingPageVisitDropped(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason pagevisitdrop.Reason,
) {
	for _, observer := range observers {
		observer.PendingPageVisitDropped(ctx, pendingPageVisit, reason)
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

func (observers PendingPageVisitObservers) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	result pagevisitresult.Result,
) {
	for _, observer := range observers {
		observer.PendingPageVisitCompleted(ctx, pendingPageVisit, result)
	}
}
