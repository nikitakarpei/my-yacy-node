package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const labelOtherWordAsks = "other_word_asks"

type discoveryRoundMetrics struct {
	unheldQueryWordsRatio      prometheusclient.Histogram
	queryWordsWithASampleRatio prometheusclient.Histogram
	partitionsPerOtherWordAsks map[wordjoined.OtherWordAsks]prometheusclient.Counter
}

func discoveryRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) discoveryRoundMetrics {
	otherWordAskPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_other_word_ask_partitions_total",
		Help: "Partitions of the ring in the word joined spreads with a sample, " +
			"by what the asks for the other query words named there.",
	}, []string{labelOtherWordAsks})
	metrics := discoveryRoundMetrics{
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words with no document in any abstract.",
		),
		queryWordsWithASampleRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_query_words_with_a_sample_ratio",
			"Share of query words with a sample.",
		),
		//exhaustive:enforce
		partitionsPerOtherWordAsks: map[wordjoined.OtherWordAsks]prometheusclient.Counter{
			wordjoined.OtherWordAsksNamingTheDocumentsToMatch: otherWordAskPartitions.WithLabelValues(
				string(wordjoined.OtherWordAsksNamingTheDocumentsToMatch),
			),
			wordjoined.OtherWordAsksOverTheCeiling: otherWordAskPartitions.WithLabelValues(
				string(wordjoined.OtherWordAsksOverTheCeiling),
			),
			wordjoined.OtherWordAsksSkipped: otherWordAskPartitions.WithLabelValues(
				string(wordjoined.OtherWordAsksSkipped),
			),
		},
	}
	registry.MustRegister(
		metrics.unheldQueryWordsRatio,
		metrics.queryWordsWithASampleRatio,
		otherWordAskPartitions,
	)

	return metrics
}

func (m discoveryRoundMetrics) observeDiscoveryRound(
	discoveryRound wordjoined.PerformedDiscoveryRound,
) {
	for _, otherWordAsks := range discoveryRound.OtherWordAsksPerPartition {
		m.partitionsPerOtherWordAsks[otherWordAsks].Inc()
	}
	if discoveryRound.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(discoveryRound.AmountOfQueryWordsHeldByNoPeer) /
			float64(discoveryRound.AmountOfQueryWords),
	)
	m.queryWordsWithASampleRatio.Observe(
		float64(discoveryRound.AmountOfQueryWordsWithASample) /
			float64(discoveryRound.AmountOfQueryWords),
	)
}
