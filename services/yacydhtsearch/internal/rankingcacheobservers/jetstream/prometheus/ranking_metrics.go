// Package prometheus reports how often the ranking cache failed.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelAction  = "action"
	actionLookup = "lookup"
	actionStore  = "store"
)

type RankingMetrics struct {
	failures *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *RankingMetrics {
	metrics := &RankingMetrics{
		failures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_ranking_cache_failures_total",
			Help: "Failures against the ranking cache, by action.",
		}, []string{labelAction}),
	}
	registry.MustRegister(metrics.failures)

	return metrics
}

func (m *RankingMetrics) RankingLookupFailed(context.Context, searchquery.Query, error) {
	m.failures.WithLabelValues(actionLookup).Inc()
}

func (m *RankingMetrics) RankingStoreFailed(context.Context, searchquery.Query, error) {
	m.failures.WithLabelValues(actionStore).Inc()
}
