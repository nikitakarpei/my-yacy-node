package websearch

import "context"

type WebSearchObserver interface {
	SearchServed(ctx context.Context, query string, searchResultCount int)
	SearchFailed(ctx context.Context, query string, cause error)
}

type WebSearchObservers []WebSearchObserver

func (observers WebSearchObservers) SearchServed(
	ctx context.Context,
	query string,
	searchResultCount int,
) {
	for _, observer := range observers {
		observer.SearchServed(ctx, query, searchResultCount)
	}
}

func (observers WebSearchObservers) SearchFailed(
	ctx context.Context,
	query string,
	cause error,
) {
	for _, observer := range observers {
		observer.SearchFailed(ctx, query, cause)
	}
}
