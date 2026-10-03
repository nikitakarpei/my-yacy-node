// Package prometheus reports how the partitions of a word joined spread were
// asked for the other words, as metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
)

const labelOtherWordAsks = "other_word_asks"

type DocumentAsksMetrics struct {
	partitionsPerKind map[documentasks.Kind]prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *DocumentAsksMetrics {
	otherWordAskPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_other_word_ask_partitions_total",
		Help: "Partitions of the ring in the word joined spreads with a leading word, " +
			"by what the asks for the other query words named there.",
	}, []string{labelOtherWordAsks})
	metrics := &DocumentAsksMetrics{
		//exhaustive:enforce
		partitionsPerKind: map[documentasks.Kind]prometheusclient.Counter{
			documentasks.NamingTheDocumentsToMatch: otherWordAskPartitions.WithLabelValues(
				string(documentasks.NamingTheDocumentsToMatch),
			),
			documentasks.OverTheCeiling: otherWordAskPartitions.WithLabelValues(
				string(documentasks.OverTheCeiling),
			),
			documentasks.Skipped: otherWordAskPartitions.WithLabelValues(
				string(documentasks.Skipped),
			),
			documentasks.PredictedOverTheCeiling: otherWordAskPartitions.WithLabelValues(
				string(documentasks.PredictedOverTheCeiling),
			),
		},
	}
	registry.MustRegister(otherWordAskPartitions)

	return metrics
}

func (m *DocumentAsksMetrics) AskedPartitionFor(
	_ context.Context,
	_ uint,
	kind documentasks.Kind,
) {
	m.partitionsPerKind[kind].Inc()
}
