package wordjoined

import "context"

type WordJoinedSpreadObserver interface {
	WordJoinedSpreadPerformed(ctx context.Context, spread PerformedWordJoinedSpread)
}

type WordJoinedSpreadObservers []WordJoinedSpreadObserver

func (observers WordJoinedSpreadObservers) WordJoinedSpreadPerformed(
	ctx context.Context,
	spread PerformedWordJoinedSpread,
) {
	for _, observer := range observers {
		observer.WordJoinedSpreadPerformed(ctx, spread)
	}
}
