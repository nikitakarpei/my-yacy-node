// Package prometheus reports how often the ranking cache answered a query as
// metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	labelResult = "result"
	resultHit   = "hit"
	resultMiss  = "miss"
)

type RankingCacheMetrics struct {
	hits   prometheusclient.Counter
	misses prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RankingCacheMetrics {
	lookups := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_ranking_cache_lookups_total",
		Help: "Queries looked up in the ranking cache, by result.",
	}, []string{labelResult})
	registry.MustRegister(lookups)

	return &RankingCacheMetrics{
		hits:   lookups.WithLabelValues(resultHit),
		misses: lookups.WithLabelValues(resultMiss),
	}
}

func (m *RankingCacheMetrics) QueryAnsweredFromCache(context.Context, searchquery.Query) {
	m.hits.Inc()
}

func (m *RankingCacheMetrics) QueryMissedCache(context.Context, searchquery.Query) {
	m.misses.Inc()
}
