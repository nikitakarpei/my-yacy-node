package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type crossCheckedDocumentsRoundMetrics struct {
	crossCheckCandidatesNoPeerTookRatio          prometheusclient.Histogram
	joinedDocumentsFoundOnlyByCrossCheckingRatio prometheusclient.Histogram
}

func crossCheckedDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) crossCheckedDocumentsRoundMetrics {
	metrics := crossCheckedDocumentsRoundMetrics{
		crossCheckCandidatesNoPeerTookRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_cross_check_candidates_no_peer_took_ratio",
			"Share of the cross-check candidates no peer took, in the spreads that had a "+
				"candidate.",
		),
		joinedDocumentsFoundOnlyByCrossCheckingRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio",
			"Share of the joined documents found only by cross-checking, in the spreads that "+
				"asked a peer to cross-check documents.",
		),
	}
	registry.MustRegister(
		metrics.crossCheckCandidatesNoPeerTookRatio,
		metrics.joinedDocumentsFoundOnlyByCrossCheckingRatio,
	)

	return metrics
}

func (m crossCheckedDocumentsRoundMetrics) observeCrossCheckedDocumentsRound(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
) {
	amountOfCandidatesNoPeerTook := crossCheckedDocumentsRound.AmountOfCrossCheckCandidatesNoPeerTook
	amountOfCandidates := crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking +
		amountOfCandidatesNoPeerTook
	if amountOfCandidates > 0 {
		m.crossCheckCandidatesNoPeerTookRatio.Observe(
			float64(amountOfCandidatesNoPeerTook) / float64(amountOfCandidates),
		)
	}
	if crossCheckedDocumentsRound.AmountOfDocumentsSentForCrossChecking == 0 ||
		crossCheckedDocumentsRound.AmountOfJoinedDocuments == 0 {
		return
	}
	m.joinedDocumentsFoundOnlyByCrossCheckingRatio.Observe(
		float64(crossCheckedDocumentsRound.AmountOfJoinedDocumentsFoundOnlyByCrossChecking) /
			float64(crossCheckedDocumentsRound.AmountOfJoinedDocuments),
	)
}
