package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type matchedAndHeldDocumentsRoundMetrics struct {
	unheldQueryWordsRatio      prometheusclient.Histogram
	fullyListedQueryWordsRatio prometheusclient.Histogram
}

func matchedAndHeldDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) matchedAndHeldDocumentsRoundMetrics {
	metrics := matchedAndHeldDocumentsRoundMetrics{
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words that no asked peer held a document for.",
		),
		fullyListedQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_fully_listed_query_words_ratio",
			"Share of query words whose peers listed every document they hold for the word.",
		),
	}
	registry.MustRegister(
		metrics.unheldQueryWordsRatio,
		metrics.fullyListedQueryWordsRatio,
	)

	return metrics
}

func (m matchedAndHeldDocumentsRoundMetrics) observeMatchedAndHeldDocumentsRound(
	matchedAndHeldDocumentsRound wordjoined.PerformedMatchedAndHeldDocumentsRound,
) {
	if matchedAndHeldDocumentsRound.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(matchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer) /
			float64(matchedAndHeldDocumentsRound.AmountOfQueryWords),
	)
	m.fullyListedQueryWordsRatio.Observe(
		float64(matchedAndHeldDocumentsRound.AmountOfFullyListedQueryWords) /
			float64(matchedAndHeldDocumentsRound.AmountOfQueryWords),
	)
}
