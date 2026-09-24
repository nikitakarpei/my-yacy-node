package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const labelOtherWordAsks = "other_word_asks"

type discoveryRoundMetrics struct {
	unheldQueryWordsRatio      prometheusclient.Histogram
	sampledQueryWordsRatio     prometheusclient.Histogram
	partitionsPerOtherWordAsks map[wordjoined.OtherWordAsks]prometheusclient.Counter
}

func discoveryRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) discoveryRoundMetrics {
	otherWordAskPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_other_word_ask_partitions_total",
		Help: "Partitions of the ring in the word joined spreads with a sampled leading word, " +
			"by what the asks for the other query words named there.",
	}, []string{labelOtherWordAsks})
	metrics := discoveryRoundMetrics{
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words with no document in any abstract.",
		),
		sampledQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_sampled_query_words_ratio",
			"Share of query words whose answers in the sampled partition gave a sample.",
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
		metrics.sampledQueryWordsRatio,
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
	m.sampledQueryWordsRatio.Observe(
		float64(discoveryRound.AmountOfSampledQueryWords) /
			float64(discoveryRound.AmountOfQueryWords),
	)
}
