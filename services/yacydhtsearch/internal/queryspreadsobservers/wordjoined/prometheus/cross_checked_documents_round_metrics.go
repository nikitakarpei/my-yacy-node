package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type crossCheckedDocumentsRoundMetrics struct {
	answeringCrossCheckedDocumentsPeersRatio                          prometheusclient.Histogram
	joinedDocumentsFoundOnlyByCrossCheckingRatio                      prometheusclient.Histogram
	leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio
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
		leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio: leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatioRegisteredIn(
			registry,
		),
	}
	registry.MustRegister(
		metrics.answeringCrossCheckedDocumentsPeersRatio,
		metrics.joinedDocumentsFoundOnlyByCrossCheckingRatio,
	)

	return metrics
}

func (m crossCheckedDocumentsRoundMetrics) observeCrossCheckedDocumentsRound(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound wordjoined.PerformedMatchedAndHeldDocumentsRound,
) {
	m.leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio.observe(
		crossCheckedDocumentsRound, matchedAndHeldDocumentsRound,
	)
	if crossCheckedDocumentsRound.AmountOfPeersAskedForCrossCheckedDocuments == 0 {
		return
	}
	m.answeringCrossCheckedDocumentsPeersRatio.Observe(
		float64(crossCheckedDocumentsRound.AmountOfPeersThatAnsweredCrossCheckedDocuments) /
			float64(crossCheckedDocumentsRound.AmountOfPeersAskedForCrossCheckedDocuments),
	)
	if crossCheckedDocumentsRound.AmountOfDocumentsJoinedWithCrossChecking == 0 {
		return
	}
	amountOfJoinedDocumentsFoundOnlyByCrossChecking := crossCheckedDocumentsRound.AmountOfDocumentsJoinedWithCrossChecking -
		crossCheckedDocumentsRound.AmountOfDocumentsJoinedWithoutCrossChecking
	m.joinedDocumentsFoundOnlyByCrossCheckingRatio.Observe(
		float64(amountOfJoinedDocumentsFoundOnlyByCrossChecking) /
			float64(crossCheckedDocumentsRound.AmountOfDocumentsJoinedWithCrossChecking),
	)
}
