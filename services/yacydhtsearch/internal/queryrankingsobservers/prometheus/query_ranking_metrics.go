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
	outcomeAnsweredByPeers   = "answered_by_peers"
	outcomeNoIndexedTerm     = "no_indexed_term"
	outcomeNoPeerToAsk       = "no_peer_to_ask"
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
	context.Context,
	searchquery.Query,
	int,
) {
	m.searches.WithLabelValues(outcomeAnsweredFromCache).Inc()
}

func (m *QueryRankingMetrics) QueryAnsweredByPeers(context.Context, searchquery.Query, int) {
	m.searches.WithLabelValues(outcomeAnsweredByPeers).Inc()
}

func (m *QueryRankingMetrics) QueryHadNoIndexedTerm(context.Context, searchquery.Query) {
	m.searches.WithLabelValues(outcomeNoIndexedTerm).Inc()
}

func (m *QueryRankingMetrics) QueryFoundNoPeerToAsk(context.Context, searchquery.Query) {
	m.searches.WithLabelValues(outcomeNoPeerToAsk).Inc()
}
