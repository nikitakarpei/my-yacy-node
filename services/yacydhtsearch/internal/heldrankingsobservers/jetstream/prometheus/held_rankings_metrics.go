// Package prometheus reports how often holding rankings in NATS failed.
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

type HeldRankingsMetrics struct {
	lookupFailures prometheusclient.Counter
	storeFailures  prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *HeldRankingsMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_ranking_cache_failures_total",
		Help: "Failures against the ranking cache, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &HeldRankingsMetrics{
		lookupFailures: failures.WithLabelValues(actionLookup),
		storeFailures:  failures.WithLabelValues(actionStore),
	}
}

func (m *HeldRankingsMetrics) RankingLookupFailed(context.Context, searchquery.Query, error) {
	m.lookupFailures.Inc()
}

func (m *HeldRankingsMetrics) RankingStoreFailed(context.Context, searchquery.Query, error) {
	m.storeFailures.Inc()
}
