package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/ratiobuckets"
)

type queryWordMetrics struct {
	unheldQueryWordsRatio prometheusclient.Histogram
}

func queryWordMetricsRegisteredIn(registry prometheusclient.Registerer) queryWordMetrics {
	metrics := queryWordMetrics{
		unheldQueryWordsRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			Help:    "Share of query words with no document in any abstract.",
			Buckets: ratiobuckets.Tenths(),
		}),
	}
	registry.MustRegister(metrics.unheldQueryWordsRatio)

	return metrics
}

func (m queryWordMetrics) observeQueryWords(spread wordjoined.PerformedWordJoinedSpread) {
	if spread.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(spread.AmountOfQueryWordsHeldByNoPeer) / float64(spread.AmountOfQueryWords),
	)
}
