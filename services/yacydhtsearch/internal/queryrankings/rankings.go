// Package queryrankings answers a query from the ranking already cached for
// it, and asks the network for a ranking only when none is cached.
package queryrankings

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

type QueryRankingObserver interface {
	QueryAnsweredFromCache(ctx context.Context, query searchquery.Query, amountOfItems int)
	QueryAnsweredByPeers(ctx context.Context, query searchquery.Query, amountOfItems int)
	QueryHoldsNoIndexedTerm(ctx context.Context, query searchquery.Query)
	QueryReachedNoPeer(ctx context.Context, query searchquery.Query)
}

type Rankings struct {
	cache    RankingCache
	network  Network
	observer QueryRankingObserver
}

func New(cache RankingCache, network Network, observer QueryRankingObserver) Rankings {
	return Rankings{cache: cache, network: network, observer: observer}
}

func (rankings Rankings) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, searchresult.Outcome) {
	if cachedRanking, cached := rankings.cache.CachedRankingFor(ctx, query); cached {
		rankings.observer.QueryAnsweredFromCache(ctx, query, len(cachedRanking.Items))

		return cachedRanking, searchresult.PeersAsked
	}

	ranking, outcome := rankings.network.Search(ctx, query)
	switch outcome {
	case searchresult.NoIndexedWordInQuery:
		rankings.observer.QueryHoldsNoIndexedTerm(ctx, query)
	case searchresult.NoPeerToAsk:
		rankings.observer.QueryReachedNoPeer(ctx, query)
	case searchresult.PeersAsked:
		rankings.cache.StoreRanking(ctx, query, ranking)
		rankings.observer.QueryAnsweredByPeers(ctx, query, len(ranking.Items))
	}

	return ranking, outcome
}
