// Package queryrankings answers a query from the ranking already cached for
// it, and asks the network for a ranking only when none is cached.
package queryrankings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type Network interface {
	Search(
		ctx context.Context,
		query searchquery.Query,
	) (searchresult.Ranking, networksearch.SearchOutcome)
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
	QueryAnsweredFromCache(ctx context.Context, query searchquery.Query, items int)
	QueryAnsweredByPeers(ctx context.Context, query searchquery.Query, items int)
	QueryHadNoIndexedTerm(ctx context.Context, query searchquery.Query)
	QueryFoundNoPeerToAsk(ctx context.Context, query searchquery.Query)
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
		r.observer.QueryAnsweredFromCache(ctx, query, len(ranking.Items))

		return ranking
	}

	return r.rankingFromPeers(ctx, query)
}

func (r Rankings) rankingFromPeers(
	ctx context.Context,
	query searchquery.Query,
) searchresult.Ranking {
	ranking, outcome := r.network.Search(ctx, query)
	switch outcome {
	case networksearch.NoIndexedTermInQuery:
		r.observer.QueryHadNoIndexedTerm(ctx, query)
	case networksearch.NoPeerToAsk:
		r.observer.QueryFoundNoPeerToAsk(ctx, query)
	case networksearch.PeersAsked:
		r.cache.StoreRanking(ctx, query, ranking)
		r.observer.QueryAnsweredByPeers(ctx, query, len(ranking.Items))
	}

	return ranking
}

type RankingObservers []RankingObserver

func (observers RankingObservers) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredFromCache(ctx, query, items)
	}
}

func (observers RankingObservers) QueryAnsweredByPeers(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	for _, observer := range observers {
		observer.QueryAnsweredByPeers(ctx, query, items)
	}
}

func (observers RankingObservers) QueryHadNoIndexedTerm(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryHadNoIndexedTerm(ctx, query)
	}
}

func (observers RankingObservers) QueryFoundNoPeerToAsk(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryFoundNoPeerToAsk(ctx, query)
	}
}
