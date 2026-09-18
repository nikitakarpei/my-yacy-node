package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type crossCheckedDocumentsRoundMetrics struct {
	documentsPastTheCrossCheckedDocumentsCeilingRatio prometheusclient.Histogram
	answeringCrossCheckedDocumentsPeersRatio          prometheusclient.Histogram
	joinedDocumentsFoundOnlyByCrossCheckingRatio      prometheusclient.Histogram
}

func crossCheckedDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) crossCheckedDocumentsRoundMetrics {
	metrics := crossCheckedDocumentsRoundMetrics{
		documentsPastTheCrossCheckedDocumentsCeilingRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_documents_past_the_cross_checked_documents_ceiling_ratio",
			"Share of the documents to cross-check that no peer could take, in the spreads that "+
				"had a document to cross-check.",
		),
		answeringCrossCheckedDocumentsPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_answering_cross_checked_documents_peers_ratio",
			"Share of the peers asked to cross-check documents that answered.",
		),
		joinedDocumentsFoundOnlyByCrossCheckingRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio",
			"Share of the joined documents found only by cross-checking, in the spreads that "+
				"asked a peer to cross-check documents.",
		),
	}
	registry.MustRegister(
		metrics.documentsPastTheCrossCheckedDocumentsCeilingRatio,
		metrics.answeringCrossCheckedDocumentsPeersRatio,
		metrics.joinedDocumentsFoundOnlyByCrossCheckingRatio,
	)

	return metrics
}

func (m crossCheckedDocumentsRoundMetrics) observeCrossCheckedDocumentsRound(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
) {
	amountOfDocumentsPastTheCeiling := crossCheckedDocumentsRound.
		AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling
	amountOfDocumentsToCrossCheck := crossCheckedDocumentsRound.
		AmountOfDocumentsSentForCrossChecking + amountOfDocumentsPastTheCeiling
	if amountOfDocumentsToCrossCheck > 0 {
		m.documentsPastTheCrossCheckedDocumentsCeilingRatio.Observe(
			float64(amountOfDocumentsPastTheCeiling) / float64(amountOfDocumentsToCrossCheck),
		)
	}
	if crossCheckedDocumentsRound.AmountOfPeersAskedForCrossCheckedDocuments == 0 {
		return
	}
	m.answeringCrossCheckedDocumentsPeersRatio.Observe(
		float64(crossCheckedDocumentsRound.AmountOfPeersThatAnsweredCrossCheckedDocuments) /
			float64(crossCheckedDocumentsRound.AmountOfPeersAskedForCrossCheckedDocuments),
	)
	if crossCheckedDocumentsRound.AmountOfJoinedDocuments == 0 {
		return
	}
	m.joinedDocumentsFoundOnlyByCrossCheckingRatio.Observe(
		float64(crossCheckedDocumentsRound.AmountOfJoinedDocumentsFoundOnlyByCrossChecking) /
			float64(crossCheckedDocumentsRound.AmountOfJoinedDocuments),
	)
}
