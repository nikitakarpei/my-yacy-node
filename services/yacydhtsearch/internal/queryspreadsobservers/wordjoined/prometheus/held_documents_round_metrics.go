package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	documentBucketCeiling = 1024.0
	documentBuckets       = 8
)

type heldDocumentsRoundMetrics struct {
	answeringHeldDocumentsPeersRatio     prometheusclient.Histogram
	emptyHeldDocumentsAnswersRatio       prometheusclient.Histogram
	joinBeforeTheHeldDocumentsAsksRatio  prometheusclient.Histogram
	documentsPastTheHeldDocumentsCeiling prometheusclient.Histogram
}

func heldDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) heldDocumentsRoundMetrics {
	metrics := heldDocumentsRoundMetrics{
		answeringHeldDocumentsPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_answering_held_documents_peers_ratio",
			"Share of the peers the second round asked which of the named documents they "+
				"hold that answered.",
		),
		emptyHeldDocumentsAnswersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_empty_held_documents_answers_ratio",
			"Share of the answers to the second round that held none of the named documents.",
		),
		joinBeforeTheHeldDocumentsAsksRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_join_before_the_held_documents_asks_ratio",
			"Share of the joined documents that the first round alone already proved. "+
				"The second round added the rest.",
		),
		documentsPastTheHeldDocumentsCeiling: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_documents_past_the_held_documents_ceiling",
				Help: "Documents of the query word with the fewest that one second round ask " +
					"had no room for.",
				Buckets: bucketsFromNoneTo(documentBucketCeiling, documentBuckets),
			},
		),
	}
	registry.MustRegister(
		metrics.answeringHeldDocumentsPeersRatio,
		metrics.emptyHeldDocumentsAnswersRatio,
		metrics.joinBeforeTheHeldDocumentsAsksRatio,
		metrics.documentsPastTheHeldDocumentsCeiling,
	)

	return metrics
}

func (m heldDocumentsRoundMetrics) observeHeldDocumentsRound(
	round wordjoined.PerformedHeldDocumentsRound,
) {
	m.documentsPastTheHeldDocumentsCeiling.Observe(
		float64(round.AmountOfDocumentsPastTheHeldDocumentsCeiling),
	)
	if round.AmountOfPeersAskedForHeldDocuments > 0 {
		m.answeringHeldDocumentsPeersRatio.Observe(
			float64(round.AmountOfPeersThatAnsweredHeldDocuments) /
				float64(round.AmountOfPeersAskedForHeldDocuments),
		)
	}
	if round.AmountOfPeersThatAnsweredHeldDocuments > 0 {
		m.emptyHeldDocumentsAnswersRatio.Observe(
			float64(round.AmountOfEmptyHeldDocumentsAnswers) /
				float64(round.AmountOfPeersThatAnsweredHeldDocuments),
		)
	}
	if round.AmountOfJoinedDocuments == 0 {
		return
	}
	m.joinBeforeTheHeldDocumentsAsksRatio.Observe(
		float64(round.AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks) /
			float64(round.AmountOfJoinedDocuments),
	)
}
