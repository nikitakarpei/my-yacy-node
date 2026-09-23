package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type crossCheckRoundMetrics struct {
	crossCheckCandidatesNoPeerTookRatio          prometheusclient.Histogram
	joinedDocumentsFoundOnlyByCrossCheckingRatio prometheusclient.Histogram
}

func crossCheckRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) crossCheckRoundMetrics {
	metrics := crossCheckRoundMetrics{
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

func (m crossCheckRoundMetrics) observeCrossCheckRound(
	crossCheckRound wordjoined.PerformedCrossCheckRound,
) {
	amountOfCandidatesNoPeerTook := crossCheckRound.AmountOfCrossCheckCandidatesNoPeerTook
	amountOfCandidates := crossCheckRound.AmountOfDocumentsSentForCrossChecking +
		amountOfCandidatesNoPeerTook
	if amountOfCandidates > 0 {
		m.crossCheckCandidatesNoPeerTookRatio.Observe(
			float64(amountOfCandidatesNoPeerTook) / float64(amountOfCandidates),
		)
	}
	if crossCheckRound.AmountOfDocumentsSentForCrossChecking == 0 ||
		crossCheckRound.AmountOfJoinedDocuments == 0 {
		return
	}
	m.joinedDocumentsFoundOnlyByCrossCheckingRatio.Observe(
		float64(crossCheckRound.AmountOfJoinedDocumentsFoundOnlyByCrossChecking) /
			float64(crossCheckRound.AmountOfJoinedDocuments),
	)
}
