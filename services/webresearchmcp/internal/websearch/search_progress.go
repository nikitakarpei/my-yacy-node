package websearch

import "context"

type SearchProgress interface {
	SearchServed(ctx context.Context, query string, searchResultCount int)
	SearchFailed(ctx context.Context, query string, cause error)
}

type SearchProgressObservers []SearchProgress

func (observers SearchProgressObservers) SearchServed(
	ctx context.Context,
	query string,
	searchResultCount int,
) {
	for _, observer := range observers {
		observer.SearchServed(ctx, query, searchResultCount)
	}
}

func (observers SearchProgressObservers) SearchFailed(
	ctx context.Context,
	query string,
	cause error,
) {
	for _, observer := range observers {
		observer.SearchFailed(ctx, query, cause)
	}
}
