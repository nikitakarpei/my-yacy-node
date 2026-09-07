// Package prometheus reports how often a cached ranking answered a query and
// how often the ranking cache failed.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelSource   = "source"
	sourceCache   = "cache"
	sourceNetwork = "network"
	labelAction   = "action"
	actionLookup  = "lookup"
	actionStore   = "store"
)

type RankingMetrics struct {
	lookups  *prometheusclient.CounterVec
	failures *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *RankingMetrics {
	metrics := &RankingMetrics{
		lookups: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_ranking_lookups_total",
			Help: "Rankings answered to a query, by where the ranking came from.",
		}, []string{labelSource}),
		failures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_ranking_cache_failures_total",
			Help: "Failures against the ranking cache, by action.",
		}, []string{labelAction}),
	}
	registry.MustRegister(metrics.lookups, metrics.failures)

	return metrics
}

func (m *RankingMetrics) CachedRankingAnswered(_ context.Context, _ searchquery.Query, _ int) {
	m.lookups.WithLabelValues(sourceCache).Inc()
}

func (m *RankingMetrics) NetworkRankingAnswered(_ context.Context, _ searchquery.Query, _ int) {
	m.lookups.WithLabelValues(sourceNetwork).Inc()
}

func (m *RankingMetrics) RankingLookupFailed(context.Context, searchquery.Query, error) {
	m.failures.WithLabelValues(actionLookup).Inc()
}

func (m *RankingMetrics) RankingStoreFailed(context.Context, searchquery.Query, error) {
	m.failures.WithLabelValues(actionStore).Inc()
}
