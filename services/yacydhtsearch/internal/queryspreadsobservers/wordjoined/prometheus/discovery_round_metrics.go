package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const labelOtherWordsAsking = "other_words_asking"

type discoveryRoundMetrics struct {
	unheldQueryWordsRatio         prometheusclient.Histogram
	sampledQueryWordsRatio        prometheusclient.Histogram
	partitionsPerOtherWordsAsking map[wordjoined.OtherWordsAsking]prometheusclient.Counter
}

func discoveryRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) discoveryRoundMetrics {
	partitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_partitions_total",
		Help: "Partitions of the ring in the word joined spreads, by how the spread asked " +
			"the other query words there.",
	}, []string{labelOtherWordsAsking})
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
		partitionsPerOtherWordsAsking: map[wordjoined.OtherWordsAsking]prometheusclient.Counter{
			wordjoined.OtherWordsAskedForTheCandidates: partitions.WithLabelValues(
				string(wordjoined.OtherWordsAskedForTheCandidates),
			),
			wordjoined.OtherWordsNotAsked: partitions.WithLabelValues(
				string(wordjoined.OtherWordsNotAsked),
			),
			wordjoined.OtherWordsAskedOverTheCeiling: partitions.WithLabelValues(
				string(wordjoined.OtherWordsAskedOverTheCeiling),
			),
			wordjoined.OtherWordsAskedWithoutASample: partitions.WithLabelValues(
				string(wordjoined.OtherWordsAskedWithoutASample),
			),
		},
	}
	registry.MustRegister(
		metrics.unheldQueryWordsRatio,
		metrics.sampledQueryWordsRatio,
		partitions,
	)

	return metrics
}

func (m discoveryRoundMetrics) observeDiscoveryRound(
	discoveryRound wordjoined.PerformedDiscoveryRound,
) {
	for _, otherWordsAsking := range discoveryRound.OtherWordsAskingPerPartition {
		m.partitionsPerOtherWordsAsking[otherWordsAsking].Inc()
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
