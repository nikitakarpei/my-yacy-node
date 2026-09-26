// Package rankingcache answers a repeated query from the ranking it holds for
// it, and asks the network only when it holds none.
package rankingcache

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type Network interface {
	Search(
		ctx context.Context,
		query searchquery.Query,
	) (searchresult.Ranking, searchresult.Outcome)
}

type HeldRankings interface {
	RankingFor(
		ctx context.Context,
		query searchquery.Query,
	) (searchresult.Ranking, bool)
	Store(
		ctx context.Context,
		query searchquery.Query,
		ranking searchresult.Ranking,
	)
}

type RankingCacheObserver interface {
	QueryAnsweredFromCache(ctx context.Context, query searchquery.Query, amountOfItems int)
	QueryAnsweredByPeers(ctx context.Context, query searchquery.Query, amountOfItems int)
	QueryHoldsNoIndexedTerm(ctx context.Context, query searchquery.Query)
	QueryReachedNoPeer(ctx context.Context, query searchquery.Query)
}

type RankingCache struct {
	heldRankings HeldRankings
	network      Network
	observer     RankingCacheObserver
}

func New(
	heldRankings HeldRankings,
	network Network,
	observer RankingCacheObserver,
) RankingCache {
	return RankingCache{heldRankings: heldRankings, network: network, observer: observer}
}

func (cache RankingCache) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, searchresult.Outcome) {
	if heldRanking, held := cache.heldRankings.RankingFor(ctx, query); held {
		cache.observer.QueryAnsweredFromCache(ctx, query, len(heldRanking.Items))

		return heldRanking, searchresult.PeersAsked
	}

	ranking, outcome := cache.network.Search(ctx, query)
	switch outcome {
	case searchresult.NoIndexedWordInQuery:
		cache.observer.QueryHoldsNoIndexedTerm(ctx, query)
	case searchresult.NoPeerToAsk:
		cache.observer.QueryReachedNoPeer(ctx, query)
	case searchresult.PeersAsked:
		cache.heldRankings.Store(ctx, query, ranking)
		cache.observer.QueryAnsweredByPeers(ctx, query, len(ranking.Items))
	}

	return ranking, outcome
}
