package rankingcache

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type RankingCacheObservers []RankingCacheObserver

func (observers RankingCacheObservers) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredFromCache(ctx, query, amountOfItems)
	}
}

func (observers RankingCacheObservers) QueryAnsweredByPeers(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredByPeers(ctx, query, amountOfItems)
	}
}

func (observers RankingCacheObservers) QueryHoldsNoIndexedTerm(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryHoldsNoIndexedTerm(ctx, query)
	}
}

func (observers RankingCacheObservers) QueryReachedNoPeer(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryReachedNoPeer(ctx, query)
	}
}
