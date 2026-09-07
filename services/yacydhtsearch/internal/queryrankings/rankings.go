// Package queryrankings answers a query from the ranking already cached for
// it, and asks the network for a ranking only when none is cached.
package queryrankings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type Network interface {
	Search(ctx context.Context, query searchquery.Query) searchresult.Ranking
}

type RankingCache interface {
	CachedRankingFor(
		ctx context.Context,
		query searchquery.Query,
	) (searchresult.Ranking, bool)
	StoreRanking(
		ctx context.Context,
		query searchquery.Query,
		ranking searchresult.Ranking,
	)
}

type RankingObserver interface {
	CachedRankingAnswered(ctx context.Context, query searchquery.Query, items int)
	NetworkRankingAnswered(ctx context.Context, query searchquery.Query, items int)
}

type Rankings struct {
	cache    RankingCache
	network  Network
	observer RankingObserver
}

func New(cache RankingCache, network Network, observer RankingObserver) Rankings {
	return Rankings{cache: cache, network: network, observer: observer}
}

func (r Rankings) RankingFor(
	ctx context.Context,
	query searchquery.Query,
) searchresult.Ranking {
	if ranking, cached := r.cache.CachedRankingFor(ctx, query); cached {
		r.observer.CachedRankingAnswered(ctx, query, len(ranking.Items))

		return ranking
	}

	ranking := r.network.Search(ctx, query)
	r.cache.StoreRanking(ctx, query, ranking)
	r.observer.NetworkRankingAnswered(ctx, query, len(ranking.Items))

	return ranking
}

type RankingObservers []RankingObserver

func (observers RankingObservers) CachedRankingAnswered(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	for _, observer := range observers {
		observer.CachedRankingAnswered(ctx, query, items)
	}
}

func (observers RankingObservers) NetworkRankingAnswered(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	for _, observer := range observers {
		observer.NetworkRankingAnswered(ctx, query, items)
	}
}
