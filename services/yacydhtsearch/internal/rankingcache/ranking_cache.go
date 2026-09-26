// Package rankingcache answers a repeated query from the ranking cached for
// it, and asks the network only when none is cached.
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

type CachedRankings interface {
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
	cachedRankings CachedRankings
	network        Network
	observer       RankingCacheObserver
}

func New(
	cachedRankings CachedRankings,
	network Network,
	observer RankingCacheObserver,
) RankingCache {
	return RankingCache{cachedRankings: cachedRankings, network: network, observer: observer}
}

func (cache RankingCache) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, searchresult.Outcome) {
	if cachedRanking, cached := cache.cachedRankings.RankingFor(ctx, query); cached {
		cache.observer.QueryAnsweredFromCache(ctx, query, len(cachedRanking.Items))

		return cachedRanking, searchresult.PeersAsked
	}

	ranking, outcome := cache.network.Search(ctx, query)
	switch outcome {
	case searchresult.NoIndexedWordInQuery:
		cache.observer.QueryHoldsNoIndexedTerm(ctx, query)
	case searchresult.NoPeerToAsk:
		cache.observer.QueryReachedNoPeer(ctx, query)
	case searchresult.PeersAsked:
		cache.cachedRankings.Store(ctx, query, ranking)
		cache.observer.QueryAnsweredByPeers(ctx, query, len(ranking.Items))
	}

	return ranking, outcome
}
