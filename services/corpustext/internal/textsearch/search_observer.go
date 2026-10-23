package textsearch

import (
	"context"
	"time"
)

type SearchObserver interface {
	SearchAnswered(
		ctx context.Context,
		search Search,
		elapsed time.Duration,
		amountOfMatches int,
	)
	SearchFailed(
		ctx context.Context,
		search Search,
		elapsed time.Duration,
		cause error,
	)
}

type SearchObservers []SearchObserver

func (observers SearchObservers) SearchAnswered(
	ctx context.Context,
	search Search,
	elapsed time.Duration,
	amountOfMatches int,
) {
	for _, observer := range observers {
		observer.SearchAnswered(ctx, search, elapsed, amountOfMatches)
	}
}

func (observers SearchObservers) SearchFailed(
	ctx context.Context,
	search Search,
	elapsed time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.SearchFailed(ctx, search, elapsed, cause)
	}
}
