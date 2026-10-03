package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type queryWordMetrics struct {
	unheldQueryWordsRatio prometheusclient.Histogram
}

func queryWordMetricsRegisteredIn(registry prometheusclient.Registerer) queryWordMetrics {
	metrics := queryWordMetrics{
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words with no document in any abstract.",
		),
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
