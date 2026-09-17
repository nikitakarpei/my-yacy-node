// Package prometheus reports how many peers the first round of a word joined
// spread asked and how many answered, how much of the query the network held,
// how often no document held all query words, how many documents the peers hold
// for a query word, how much of the join an item of an answer already covers
// and how much of that carried a posting, and how much of the join the second
// round asked metadata for and got back, as metrics.
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
	wordJoinedSpreads                   *prometheusclient.CounterVec
	peersAsked                          prometheusclient.Histogram
	answeringPeersRatio                 prometheusclient.Histogram
	unheldQueryWordsRatio               prometheusclient.Histogram
	joinWithMetadataRatio               prometheusclient.Histogram
	matchedDocumentsCountedByAPeerRatio prometheusclient.Histogram
	missingMetadataAskedForRatio        prometheusclient.Histogram
	askedDocumentsWithMetadataRatio     prometheusclient.Histogram
	wordJoinedSpreadDurationSeconds     prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSpreadMetrics {
	metrics := &WordJoinedSpreadMetrics{
		wordJoinedSpreads: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_word_joined_spreads_total",
			Help: "Word joined spreads, by what the first round found for all query words.",
		}, []string{labelJoin}),
		peersAsked: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_peers_asked",
				Help: "Peers the first round of a word joined spread asked which documents " +
					"they hold. A peer responsible for several query words counts once.",
				Buckets: bucketsFromNoneTo(peerBucketCeiling, peerBuckets),
			},
		),
		answeringPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_answering_peers_ratio",
			"Share of the peers asked which documents they hold that answered.",
		),
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_unheld_query_words_ratio",
			"Share of query words that no asked peer held a document for.",
		),
		joinWithMetadataRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_join_with_metadata_ratio",
			"Share of the joined documents that an answer to the first round already "+
				"carried the metadata of.",
		),
		matchedDocumentsCountedByAPeerRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_matched_documents_counted_by_a_peer_ratio",
			"Share of the documents the peers matched in their answers to the first round "+
				"that a peer counted a word in.",
		),
		missingMetadataAskedForRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_missing_metadata_asked_for_ratio",
			"Share of the joined documents that came without metadata the spread "+
				"asked the peers metadata for.",
		),
		askedDocumentsWithMetadataRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_asked_documents_with_metadata_ratio",
			"Share of the documents the spread asked metadata for that came back with metadata.",
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
		metrics.peersAsked,
		metrics.answeringPeersRatio,
		metrics.unheldQueryWordsRatio,
		metrics.joinWithMetadataRatio,
		metrics.matchedDocumentsCountedByAPeerRatio,
		metrics.missingMetadataAskedForRatio,
		metrics.askedDocumentsWithMetadataRatio,
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
	m.peersAsked.Observe(float64(spread.AmountOfPeersAsked))
	m.wordJoinedSpreadDurationSeconds.Observe(spread.TimeSpent.Seconds())
	m.countJoin(spread)
	m.observeAnsweringPeersRatio(spread)
	m.observeQueryRatios(spread)
	m.observeMatchedDocumentsCountedByAPeerRatio(spread)
}

func (m *WordJoinedSpreadMetrics) observeMatchedDocumentsCountedByAPeerRatio(
	spread wordjoined.PerformedWordJoinedSpread,
) {
	if spread.AmountOfMatchedDocumentsAcrossAnswers == 0 {
		return
	}
	m.matchedDocumentsCountedByAPeerRatio.Observe(
		float64(spread.AmountOfMatchedDocumentsCountedByAPeer) /
			float64(spread.AmountOfMatchedDocumentsAcrossAnswers),
	)
}

func (m *WordJoinedSpreadMetrics) countJoin(spread wordjoined.PerformedWordJoinedSpread) {
	if spread.AmountOfJoinedDocuments == 0 {
		m.wordJoinedSpreads.WithLabelValues(joinFoundNoDocument).Inc()

		return
	}
	m.wordJoinedSpreads.WithLabelValues(joinFoundDocuments).Inc()
	m.joinWithMetadataRatio.Observe(
		float64(spread.AmountOfJoinedDocumentsWithMetadata) /
			float64(spread.AmountOfJoinedDocuments),
	)
	documentsMissingMetadata := spread.AmountOfJoinedDocuments -
		spread.AmountOfJoinedDocumentsWithMetadata
	if documentsMissingMetadata == 0 {
		return
	}
	m.missingMetadataAskedForRatio.Observe(
		float64(spread.AmountOfDocumentsAskedMetadataFor) / float64(documentsMissingMetadata),
	)
}

func (m *WordJoinedSpreadMetrics) observeAnsweringPeersRatio(
	spread wordjoined.PerformedWordJoinedSpread,
) {
	if spread.AmountOfPeersAsked == 0 {
		return
	}
	m.answeringPeersRatio.Observe(
		float64(spread.AmountOfPeersThatAnswered) / float64(spread.AmountOfPeersAsked),
	)
}

func (m *WordJoinedSpreadMetrics) observeQueryRatios(
	spread wordjoined.PerformedWordJoinedSpread,
) {
	if spread.AmountOfQueryWords > 0 {
		m.unheldQueryWordsRatio.Observe(
			float64(spread.AmountOfQueryWordsHeldByNoPeer) / float64(spread.AmountOfQueryWords),
		)
	}
	if spread.AmountOfDocumentsAskedMetadataFor == 0 {
		return
	}
	m.askedDocumentsWithMetadataRatio.Observe(
		float64(spread.AmountOfAskedDocumentsWithMetadata) /
			float64(spread.AmountOfDocumentsAskedMetadataFor),
	)
}
