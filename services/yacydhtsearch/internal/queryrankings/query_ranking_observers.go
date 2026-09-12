package queryrankings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type QueryRankingObservers []QueryRankingObserver

func (observers QueryRankingObservers) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredFromCache(ctx, query, amountOfItems)
	}
}

func (observers QueryRankingObservers) QueryAnsweredByPeers(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredByPeers(ctx, query, amountOfItems)
	}
}

func (observers QueryRankingObservers) QueryHoldsNoIndexedTerm(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryHoldsNoIndexedTerm(ctx, query)
	}
}

func (observers QueryRankingObservers) QueryReachedNoPeer(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryReachedNoPeer(ctx, query)
	}
}
