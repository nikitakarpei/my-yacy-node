// Package prometheus reports the outcome each query reached as metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelOutcome             = "outcome"
	outcomeAnsweredFromCache = "answered_from_cache"
	outcomeNoItemFromCache   = "no_item_from_cache"
	outcomeAnsweredByPeers   = "answered_by_peers"
	outcomeNoItemFromPeers   = "no_item_from_peers"
	outcomeNoIndexedTerm     = "no_indexed_term"
	outcomeNoPeerReached     = "no_peer_reached"
)

type QueryRankingMetrics struct {
	searches *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *QueryRankingMetrics {
	metrics := &QueryRankingMetrics{
		searches: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_searches_total",
			Help: "Searches answered, by the outcome each search reached.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.searches)

	return metrics
}

func (m *QueryRankingMetrics) QueryAnsweredFromCache(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	m.countSearch(outcomeAnsweredFromCache, outcomeNoItemFromCache, amountOfItems)
}

func (m *QueryRankingMetrics) QueryAnsweredByPeers(
	_ context.Context,
	_ searchquery.Query,
	amountOfItems int,
) {
	m.countSearch(outcomeAnsweredByPeers, outcomeNoItemFromPeers, amountOfItems)
}

func (m *QueryRankingMetrics) countSearch(
	outcomeWithItems string,
	outcomeWithNoItem string,
	amountOfItems int,
) {
	if amountOfItems == 0 {
		m.searches.WithLabelValues(outcomeWithNoItem).Inc()

		return
	}
	m.searches.WithLabelValues(outcomeWithItems).Inc()
}

func (m *QueryRankingMetrics) QueryHoldsNoIndexedTerm(context.Context, searchquery.Query) {
	m.searches.WithLabelValues(outcomeNoIndexedTerm).Inc()
}

func (m *QueryRankingMetrics) QueryReachedNoPeer(context.Context, searchquery.Query) {
	m.searches.WithLabelValues(outcomeNoPeerReached).Inc()
}
