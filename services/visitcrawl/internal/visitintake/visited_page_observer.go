package visitintake

import "context"

type VisitedPageObserver interface {
	VisitedPageRedirected(ctx context.Context, visitedPage string)
	VisitedPageRejected(ctx context.Context, cause error)
	VisitedPageMethodRefused(ctx context.Context, method string)
}

type VisitedPageObservers []VisitedPageObserver

func (observers VisitedPageObservers) VisitedPageRedirected(
	ctx context.Context,
	visitedPage string,
) {
	for _, observer := range observers {
		observer.VisitedPageRedirected(ctx, visitedPage)
	}
}

func (observers VisitedPageObservers) VisitedPageRejected(ctx context.Context, cause error) {
	for _, observer := range observers {
		observer.VisitedPageRejected(ctx, cause)
	}
}

func (observers VisitedPageObservers) VisitedPageMethodRefused(
	ctx context.Context,
	method string,
) {
	for _, observer := range observers {
		observer.VisitedPageMethodRefused(ctx, method)
	}
}
