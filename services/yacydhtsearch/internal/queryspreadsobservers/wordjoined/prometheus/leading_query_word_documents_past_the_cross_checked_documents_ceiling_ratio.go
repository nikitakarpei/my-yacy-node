package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio map[wordjoined.LeadingQueryWordStanding]prometheusclient.Observer

func leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatioRegisteredIn(
	registry prometheusclient.Registerer,
) leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio {
	leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatios := prometheusclient.NewHistogramVec(
		prometheusclient.HistogramOpts{
			Name: "yacydhtsearch_word_joined_spread_leading_query_word_documents_past_the_cross_checked_documents_ceiling_ratio",
			Help: "Share of the documents listed for the leading query word that were past the " +
				"cross-checked documents ceiling, by which query word led the join.",
			Buckets: prometheusclient.LinearBuckets(0, ratioBucketWidth, amountOfRatioBuckets),
		},
		[]string{labelLeadingQueryWordStanding},
	)
	registry.MustRegister(leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatios)

	//exhaustive:enforce
	leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio := leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio{
		wordjoined.RarestFullyListedQueryWord: leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatios.WithLabelValues(
			string(wordjoined.RarestFullyListedQueryWord),
		),
		wordjoined.MoreCommonFullyListedQueryWord: leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatios.WithLabelValues(
			string(wordjoined.MoreCommonFullyListedQueryWord),
		),
		wordjoined.RarestPartlyListedQueryWord: leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatios.WithLabelValues(
			string(wordjoined.RarestPartlyListedQueryWord),
		),
	}

	return leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio
}

func (ratio leadingQueryWordDocumentsPastTheCrossCheckedDocumentsCeilingRatio) observe(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound wordjoined.PerformedMatchedAndHeldDocumentsRound,
) {
	if matchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord == 0 {
		return
	}
	ratio[matchedAndHeldDocumentsRound.LeadingQueryWordStanding].Observe(
		float64(crossCheckedDocumentsRound.AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling) /
			float64(
				matchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord,
			),
	)
}
