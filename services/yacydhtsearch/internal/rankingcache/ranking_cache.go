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
	) (searchresult.Ranking, bool)
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
	QueryAnsweredFromCache(ctx context.Context, query searchquery.Query)
	QueryMissedCache(ctx context.Context, query searchquery.Query)
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
) (searchresult.Ranking, bool) {
	if cachedRanking, cached := cache.cachedRankings.RankingFor(ctx, query); cached {
		cache.observer.QueryAnsweredFromCache(ctx, query)

		return cachedRanking, true
	}
	cache.observer.QueryMissedCache(ctx, query)

	ranking, ranked := cache.network.Search(ctx, query)
	if ranked {
		cache.cachedRankings.Store(ctx, query, ranking)
	}

	return ranking, ranked
}
