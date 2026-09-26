// Package prometheus reports how often caching rankings in NATS failed.
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

type CachedRankingsMetrics struct {
	lookupFailures prometheusclient.Counter
	storeFailures  prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *CachedRankingsMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_ranking_cache_failures_total",
		Help: "Failures against the ranking cache, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &CachedRankingsMetrics{
		lookupFailures: failures.WithLabelValues(actionLookup),
		storeFailures:  failures.WithLabelValues(actionStore),
	}
}

func (m *CachedRankingsMetrics) RankingLookupFailed(context.Context, searchquery.Query, error) {
	m.lookupFailures.Inc()
}

func (m *CachedRankingsMetrics) RankingStoreFailed(context.Context, searchquery.Query, error) {
	m.storeFailures.Inc()
}
