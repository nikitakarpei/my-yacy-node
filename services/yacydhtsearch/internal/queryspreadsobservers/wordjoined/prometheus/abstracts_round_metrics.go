package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type abstractsRoundMetrics struct {
	unheldQueryWordsRatio                prometheusclient.Histogram
	queryWordsWithCompleteAbstractsRatio prometheusclient.Histogram
}

func abstractsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) abstractsRoundMetrics {
	metrics := abstractsRoundMetrics{
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words that no asked peer held a document for.",
		),
		queryWordsWithCompleteAbstractsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_query_words_with_complete_abstracts_ratio",
			"Share of query words whose peers sent complete abstracts.",
		),
	}
	registry.MustRegister(
		metrics.unheldQueryWordsRatio,
		metrics.queryWordsWithCompleteAbstractsRatio,
	)

	return metrics
}

func (m abstractsRoundMetrics) observeAbstractsRound(
	abstractsRound wordjoined.PerformedAbstractsRound,
) {
	if abstractsRound.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(abstractsRound.AmountOfQueryWordsHeldByNoPeer) /
			float64(abstractsRound.AmountOfQueryWords),
	)
	m.queryWordsWithCompleteAbstractsRatio.Observe(
		float64(abstractsRound.AmountOfQueryWordsWithCompleteAbstracts) /
			float64(abstractsRound.AmountOfQueryWords),
	)
}
