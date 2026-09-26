// Package prometheus reports the outcome each query reached as metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelOutcome             = "outcome"
	outcomeAnsweredFromCache = "answered from cache"
	outcomeNoItemFromCache   = "no item from cache"
	outcomeAnsweredByPeers   = "answered by peers"
	outcomeNoItemFromPeers   = "no item from peers"
	outcomeNoIndexedTerm     = "no indexed term"
	outcomeNoPeerReached     = "no peer reached"
)

type RankingCacheMetrics struct {
	searchesAnsweredFromCache   prometheusclient.Counter
	searchesWithNoItemFromCache prometheusclient.Counter
	searchesAnsweredByPeers     prometheusclient.Counter
	searchesWithNoItemFromPeers prometheusclient.Counter
	searchesWithNoIndexedTerm   prometheusclient.Counter
	searchesThatReachedNoPeer   prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RankingCacheMetrics {
	searches := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_searches_total",
		Help: "Searches answered, by the outcome each search reached.",
	}, []string{labelOutcome})
	registry.MustRegister(searches)

	return &RankingCacheMetrics{
		searchesAnsweredFromCache:   searches.WithLabelValues(outcomeAnsweredFromCache),
		searchesWithNoItemFromCache: searches.WithLabelValues(outcomeNoItemFromCache),
		searchesAnsweredByPeers:     searches.WithLabelValues(outcomeAnsweredByPeers),
		searchesWithNoItemFromPeers: searches.WithLabelValues(outcomeNoItemFromPeers),
		searchesWithNoIndexedTerm:   searches.WithLabelValues(outcomeNoIndexedTerm),
		searchesThatReachedNoPeer:   searches.WithLabelValues(outcomeNoPeerReached),
	}
}

func (m *RankingCacheMetrics) QueryAnsweredFromCache(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	if amountOfItems == 0 {
		m.searchesWithNoItemFromCache.Inc()

		return
	}
	m.searchesAnsweredFromCache.Inc()
}

func (m *RankingCacheMetrics) QueryAnsweredByPeers(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	if amountOfItems == 0 {
		m.searchesWithNoItemFromPeers.Inc()

		return
	}
	m.searchesAnsweredByPeers.Inc()
}

func (m *RankingCacheMetrics) QueryHoldsNoIndexedTerm(context.Context, searchquery.Query) {
	m.searchesWithNoIndexedTerm.Inc()
}

func (m *RankingCacheMetrics) QueryReachedNoPeer(context.Context, searchquery.Query) {
	m.searchesThatReachedNoPeer.Inc()
}
