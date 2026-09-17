package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type crossCheckedDocumentsRoundMetrics struct {
	answeringCrossCheckedDocumentsPeersRatio                       prometheusclient.Histogram
	joinedDocumentsFoundOnlyByCrossCheckingRatio                   prometheusclient.Histogram
	documentsListedForTheRarestQueryWordThatMissedACrossCheckRatio prometheusclient.Histogram
}

func crossCheckedDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) crossCheckedDocumentsRoundMetrics {
	metrics := crossCheckedDocumentsRoundMetrics{
		answeringCrossCheckedDocumentsPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_answering_cross_checked_documents_peers_ratio",
			"Share of the peers asked to cross-check documents that answered.",
		),
		joinedDocumentsFoundOnlyByCrossCheckingRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio",
			"Share of the joined documents found only by cross-checking, in the spreads that "+
				"asked a peer to cross-check documents.",
		),
		documentsListedForTheRarestQueryWordThatMissedACrossCheckRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_documents_listed_for_the_rarest_query_word_that_missed_a_cross_check_ratio",
			"Share of the documents listed for the rarest query word that missed a cross-check.",
		),
	}
	registry.MustRegister(
		metrics.answeringCrossCheckedDocumentsPeersRatio,
		metrics.joinedDocumentsFoundOnlyByCrossCheckingRatio,
		metrics.documentsListedForTheRarestQueryWordThatMissedACrossCheckRatio,
	)

	return metrics
}

func (m crossCheckedDocumentsRoundMetrics) observeCrossCheckedDocumentsRound(
	round wordjoined.PerformedCrossCheckedDocumentsRound,
	amountOfDocumentsListedByThePeersOfTheRarestQueryWord int,
) {
	if amountOfDocumentsListedByThePeersOfTheRarestQueryWord > 0 {
		m.documentsListedForTheRarestQueryWordThatMissedACrossCheckRatio.Observe(
			float64(round.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling) /
				float64(amountOfDocumentsListedByThePeersOfTheRarestQueryWord),
		)
	}
	if round.AmountOfPeersAskedForCrossCheckedDocuments == 0 {
		return
	}
	m.answeringCrossCheckedDocumentsPeersRatio.Observe(
		float64(round.AmountOfPeersThatAnsweredCrossCheckedDocuments) /
			float64(round.AmountOfPeersAskedForCrossCheckedDocuments),
	)
	if round.AmountOfJoinedDocuments == 0 {
		return
	}
	amountOfJoinedDocumentsFoundOnlyByAsking := round.AmountOfJoinedDocuments -
		round.AmountOfJoinedDocumentsBeforeTheCrossCheckedDocumentsAsks
	m.joinedDocumentsFoundOnlyByCrossCheckingRatio.Observe(
		float64(amountOfJoinedDocumentsFoundOnlyByAsking) / float64(round.AmountOfJoinedDocuments),
	)
}
