package wordjoined

import "context"

type WordJoinedSearchObserver interface {
	WordJoinedSearchPerformed(ctx context.Context, search PerformedWordJoinedSearch)
}

type WordJoinedSearchObservers []WordJoinedSearchObserver

func (observers WordJoinedSearchObservers) WordJoinedSearchPerformed(
	ctx context.Context,
	search PerformedWordJoinedSearch,
) {
	for _, observer := range observers {
		observer.WordJoinedSearchPerformed(ctx, search)
	}
}
