// Package prometheus reports how a word joined spread performed, as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	peerBucketCeiling   = 128.0
	peerBuckets         = 8
	durationBuckets     = 12
	budgetShare         = 1024
	ratioBuckets        = 11
	ratioStep           = 0.1
	labelJoin           = "join"
	joinFoundNoDocument = "no document"
	joinFoundDocuments  = "documents"
)

type WordJoinedSpreadMetrics struct {
	wordJoinedSpreads                               *prometheusclient.CounterVec
	peersAskedForMatchedAndHeldDocuments            prometheusclient.Histogram
	answeringMatchedAndHeldDocumentsPeersRatio      prometheusclient.Histogram
	unheldQueryWordsRatio                           prometheusclient.Histogram
	fullyListedQueryWordsRatio                      prometheusclient.Histogram
	crossCheckedDocumentsRound                      crossCheckedDocumentsRoundMetrics
	joinedDocumentsDroppedBeforeMetadataLookupRatio prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio           prometheusclient.Histogram
	wordJoinedSpreadDurationSeconds                 prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSpreadMetrics {
	metrics := &WordJoinedSpreadMetrics{
		wordJoinedSpreads: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_word_joined_spreads_total",
			Help: "Word joined spreads, by whether the join found a document.",
		}, []string{labelJoin}),
		peersAskedForMatchedAndHeldDocuments: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_peers_asked_for_matched_and_held_documents",
				Help: "Distinct peers a word joined spread asked which documents they hold " +
					"for a query word.",
				Buckets: bucketsFromNoneTo(peerBucketCeiling, peerBuckets),
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
		crossCheckedDocumentsRound: crossCheckedDocumentsRoundMetricsRegisteredIn(registry),
		joinedDocumentsDroppedBeforeMetadataLookupRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio",
			"Share of the joined documents without metadata that the spread dropped before "+
				"looking their metadata up.",
		),
		lookedUpDocumentsWithoutMetadataRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio",
			"Share of the documents the spread looked metadata up for that no peer sent "+
				"metadata for.",
		),
		wordJoinedSpreadDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_duration_seconds",
				Help: "Word joined spread duration in seconds.",
				Buckets: prometheusclient.ExponentialBucketsRange(
					queryBudget.Seconds()/budgetShare,
					queryBudget.Seconds()*2,
					durationBuckets,
				),
			},
		),
	}
	registry.MustRegister(
		metrics.wordJoinedSpreads,
		metrics.peersAskedForMatchedAndHeldDocuments,
		metrics.answeringMatchedAndHeldDocumentsPeersRatio,
		metrics.unheldQueryWordsRatio,
		metrics.fullyListedQueryWordsRatio,
		metrics.joinedDocumentsDroppedBeforeMetadataLookupRatio,
		metrics.lookedUpDocumentsWithoutMetadataRatio,
		metrics.wordJoinedSpreadDurationSeconds,
	)

	return metrics
}

func ratioHistogramNamed(name string, help string) prometheusclient.Histogram {
	return prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheusclient.LinearBuckets(0, ratioStep, ratioBuckets),
	})
}

func bucketsFromNoneTo(ceiling float64, buckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, buckets)...,
	)
}

func (m *WordJoinedSpreadMetrics) WordJoinedSpreadPerformed(
	_ context.Context,
	spread wordjoined.PerformedWordJoinedSpread,
) {
	m.peersAskedForMatchedAndHeldDocuments.Observe(float64(
		spread.MatchedAndHeldDocumentsRound.AmountOfPeersAskedForMatchedAndHeldDocuments,
	))
	m.wordJoinedSpreadDurationSeconds.Observe(spread.TimeSpent.Seconds())
	m.countJoin(spread.CrossCheckedDocumentsRound, spread.URLMetadataRound)
	m.observeMatchedAndHeldDocumentsRound(spread.MatchedAndHeldDocumentsRound)
	m.crossCheckedDocumentsRound.observeCrossCheckedDocumentsRound(
		spread.CrossCheckedDocumentsRound,
		spread.MatchedAndHeldDocumentsRound.AmountOfDocumentsListedByThePeersOfTheRarestQueryWord,
	)
	m.observeLookedUpDocumentsWithoutMetadataRatio(spread.URLMetadataRound)
}

func (m *WordJoinedSpreadMetrics) countJoin(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
	urlMetadataRound wordjoined.PerformedURLMetadataRound,
) {
	if crossCheckedDocumentsRound.AmountOfJoinedDocuments == 0 {
		m.wordJoinedSpreads.WithLabelValues(joinFoundNoDocument).Inc()

		return
	}
	m.wordJoinedSpreads.WithLabelValues(joinFoundDocuments).Inc()
	documentsMissingMetadata := crossCheckedDocumentsRound.AmountOfJoinedDocuments -
		urlMetadataRound.AmountOfJoinedDocumentsWithMetadata
	if documentsMissingMetadata == 0 {
		return
	}
	m.joinedDocumentsDroppedBeforeMetadataLookupRatio.Observe(
		float64(documentsMissingMetadata-urlMetadataRound.AmountOfDocumentsAskedMetadataFor) /
			float64(documentsMissingMetadata),
	)
}

func (m *WordJoinedSpreadMetrics) observeMatchedAndHeldDocumentsRound(
	round wordjoined.PerformedMatchedAndHeldDocumentsRound,
) {
	if round.AmountOfPeersAskedForMatchedAndHeldDocuments > 0 {
		m.answeringMatchedAndHeldDocumentsPeersRatio.Observe(
			float64(round.AmountOfPeersThatAnsweredMatchedAndHeldDocuments) /
				float64(round.AmountOfPeersAskedForMatchedAndHeldDocuments),
		)
	}
	if round.AmountOfQueryWords == 0 {
		return
	}
	m.unheldQueryWordsRatio.Observe(
		float64(round.AmountOfQueryWordsHeldByNoPeer) / float64(round.AmountOfQueryWords),
	)
	m.fullyListedQueryWordsRatio.Observe(
		float64(round.AmountOfFullyListedQueryWords) / float64(round.AmountOfQueryWords),
	)
}

func (m *WordJoinedSpreadMetrics) observeLookedUpDocumentsWithoutMetadataRatio(
	round wordjoined.PerformedURLMetadataRound,
) {
	if round.AmountOfDocumentsAskedMetadataFor == 0 {
		return
	}
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(round.AmountOfDocumentsAskedMetadataFor-round.AmountOfAskedDocumentsWithMetadata) /
			float64(round.AmountOfDocumentsAskedMetadataFor),
	)
}
