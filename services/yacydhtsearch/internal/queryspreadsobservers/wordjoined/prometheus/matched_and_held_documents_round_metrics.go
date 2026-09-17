package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	peerBucketCeiling   = 128.0
	amountOfPeerBuckets = 8
)

type matchedAndHeldDocumentsRoundMetrics struct {
	peersAskedForMatchedAndHeldDocuments       prometheusclient.Histogram
	answeringMatchedAndHeldDocumentsPeersRatio prometheusclient.Histogram
	unheldQueryWordsRatio                      prometheusclient.Histogram
	fullyListedQueryWordsRatio                 prometheusclient.Histogram
}

func matchedAndHeldDocumentsRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) matchedAndHeldDocumentsRoundMetrics {
	metrics := matchedAndHeldDocumentsRoundMetrics{
		peersAskedForMatchedAndHeldDocuments: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_peers_asked_for_matched_and_held_documents",
				Help: "Distinct peers a word joined spread asked which documents they hold " +
					"for a query word.",
				Buckets: bucketsFromNoneTo(peerBucketCeiling, amountOfPeerBuckets),
			},
		),
		answeringMatchedAndHeldDocumentsPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_answering_matched_and_held_documents_peers_ratio",
			"Share of the peers asked which documents they hold for a query word that answered.",
		),
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words that no asked peer held a document for.",
		),
		fullyListedQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_fully_listed_query_words_ratio",
			"Share of query words whose peers listed every document they hold for the word.",
		),
	}
	registry.MustRegister(
		metrics.peersAskedForMatchedAndHeldDocuments,
		metrics.answeringMatchedAndHeldDocumentsPeersRatio,
		metrics.unheldQueryWordsRatio,
		metrics.fullyListedQueryWordsRatio,
	)

	return metrics
}

func bucketsFromNoneTo(ceiling float64, amountOfBuckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, amountOfBuckets)...,
	)
}

func (m matchedAndHeldDocumentsRoundMetrics) observeMatchedAndHeldDocumentsRound(
	matchedAndHeldDocumentsRound wordjoined.PerformedMatchedAndHeldDocumentsRound,
) {
	m.peersAskedForMatchedAndHeldDocuments.Observe(float64(
		matchedAndHeldDocumentsRound.AmountOfPeersAskedForMatchedAndHeldDocuments,
	))
	if matchedAndHeldDocumentsRound.AmountOfPeersAskedForMatchedAndHeldDocuments > 0 {
		m.answeringMatchedAndHeldDocumentsPeersRatio.Observe(
			float64(matchedAndHeldDocumentsRound.AmountOfPeersThatAnsweredMatchedAndHeldDocuments) /
				float64(matchedAndHeldDocumentsRound.AmountOfPeersAskedForMatchedAndHeldDocuments),
		)
	}
	if matchedAndHeldDocumentsRound.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(matchedAndHeldDocumentsRound.AmountOfQueryWordsHeldByNoPeer) /
			float64(matchedAndHeldDocumentsRound.AmountOfQueryWords),
	)
	m.fullyListedQueryWordsRatio.Observe(
		float64(matchedAndHeldDocumentsRound.AmountOfFullyListedQueryWords) /
			float64(matchedAndHeldDocumentsRound.AmountOfQueryWords),
	)
}
