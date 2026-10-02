package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/matchingwords"
)

const labelMatchingWordAsks = "other_word_asks"

type matchingWordAskMetrics struct {
	partitionsPerKind map[matchingwords.Kind]prometheusclient.Counter
}

func matchingWordAskMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) matchingWordAskMetrics {
	matchingWordAskPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_other_word_ask_partitions_total",
		Help: "Partitions of the ring in the word joined spreads with a leading word, " +
			"by what the asks for the other query words named there.",
	}, []string{labelMatchingWordAsks})
	metrics := matchingWordAskMetrics{
		//exhaustive:enforce
		partitionsPerKind: map[matchingwords.Kind]prometheusclient.Counter{
			matchingwords.NamingTheDocumentsToMatch: matchingWordAskPartitions.WithLabelValues(
				string(matchingwords.NamingTheDocumentsToMatch),
			),
			matchingwords.OverTheCeiling: matchingWordAskPartitions.WithLabelValues(
				string(matchingwords.OverTheCeiling),
			),
			matchingwords.Skipped: matchingWordAskPartitions.WithLabelValues(
				string(matchingwords.Skipped),
			),
			matchingwords.PredictedOverTheCeiling: matchingWordAskPartitions.WithLabelValues(
				string(matchingwords.PredictedOverTheCeiling),
			),
		},
	}
	registry.MustRegister(matchingWordAskPartitions)

	return metrics
}

func (m matchingWordAskMetrics) observeMatchingWordAsks(
	asksPerPartition matchingwords.AsksPerPartition,
) {
	for _, kind := range asksPerPartition {
		m.partitionsPerKind[kind].Inc()
	}
}
