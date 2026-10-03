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
	partitionsPerDecision map[documentasks.DocumentsToMatchDecision]prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *DocumentAsksMetrics {
	otherWordAskPartitions := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_other_word_ask_partitions_total",
		Help: "Partitions of the ring in the word joined spreads with a leading word, " +
			"by what the asks for the other query words named there.",
	}, []string{labelOtherWordAsks})
	metrics := &DocumentAsksMetrics{
		//exhaustive:enforce
		partitionsPerDecision: map[documentasks.DocumentsToMatchDecision]prometheusclient.Counter{
			documentasks.NamedTheDocumentsToMatch: otherWordAskPartitions.WithLabelValues(
				string(documentasks.NamedTheDocumentsToMatch),
			),
			documentasks.NamedNoneOverTheCeiling: otherWordAskPartitions.WithLabelValues(
				string(documentasks.NamedNoneOverTheCeiling),
			),
			documentasks.NoDocumentsToMatch: otherWordAskPartitions.WithLabelValues(
				string(documentasks.NoDocumentsToMatch),
			),
			documentasks.NamedNonePredictedOverTheCeiling: otherWordAskPartitions.WithLabelValues(
				string(documentasks.NamedNonePredictedOverTheCeiling),
			),
		},
	}
	registry.MustRegister(otherWordAskPartitions)

	return metrics
}

func (m *DocumentAsksMetrics) AskedAmongTheDocuments(
	_ context.Context,
	decisionPerPartition documentasks.DocumentsToMatchDecisionPerPartition,
) {
	for decision, amountOfPartitions := range decisionPerPartition.AmountOfPartitionsPerDecision() {
		m.partitionsPerDecision[decision].Add(float64(amountOfPartitions))
	}
}
