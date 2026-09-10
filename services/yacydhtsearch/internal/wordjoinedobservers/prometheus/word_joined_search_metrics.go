// Package prometheus reports how many peers the first round of a word joined
// search asked and how many answered, how much of the query the network held,
// how often no document held all query words, and how much of the join the
// second round asked metadata for and got back, as metrics.
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

type WordJoinedSearchMetrics struct {
	wordJoinedSearches               *prometheusclient.CounterVec
	peersAskedForHeldDocuments       prometheusclient.Histogram
	heldDocumentsAnsweringPeersRatio prometheusclient.Histogram
	unheldQueryWordsRatio            prometheusclient.Histogram
	joinAskedMetadataForRatio        prometheusclient.Histogram
	metadataThatCameBackRatio        prometheusclient.Histogram
	wordJoinedSearchDurationSeconds  prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *WordJoinedSearchMetrics {
	metrics := &WordJoinedSearchMetrics{
		wordJoinedSearches: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_word_joined_searches_total",
			Help: "Word joined searches, by what the first round found for all query words.",
		}, []string{labelJoin}),
		peersAskedForHeldDocuments: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_search_peers_asked_for_held_documents",
				Help: "Peers the first round of a word joined search asked which documents " +
					"they hold. A peer responsible for several query words counts once.",
				Buckets: bucketsFromNoneTo(peerBucketCeiling, peerBuckets),
			},
		),
		heldDocumentsAnsweringPeersRatio: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name:    "yacydhtsearch_word_joined_search_held_documents_answering_peers_ratio",
				Help:    "Share of the peers asked which documents they hold that answered.",
				Buckets: prometheusclient.LinearBuckets(0, ratioStep, ratioBuckets),
			},
		),
		unheldQueryWordsRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_search_unheld_query_words_ratio",
			Help:    "Share of query words that no asked peer held a document for.",
			Buckets: prometheusclient.LinearBuckets(0, ratioStep, ratioBuckets),
		}),
		joinAskedMetadataForRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_search_join_asked_metadata_for_ratio",
			Help:    "Share of the joined documents the search asked the peers metadata for.",
			Buckets: prometheusclient.LinearBuckets(0, ratioStep, ratioBuckets),
		}),
		metadataThatCameBackRatio: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name: "yacydhtsearch_word_joined_search_metadata_that_came_back_ratio",
			Help: "Share of the documents the search asked metadata for that came back " +
				"as an item.",
			Buckets: prometheusclient.LinearBuckets(0, ratioStep, ratioBuckets),
		}),
		wordJoinedSearchDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_search_duration_seconds",
				Help: "Word joined search duration in seconds.",
				Buckets: prometheusclient.ExponentialBucketsRange(
					queryBudget.Seconds()/budgetShare,
					queryBudget.Seconds()*2,
					durationBuckets,
				),
			},
		),
	}
	registry.MustRegister(
		metrics.wordJoinedSearches,
		metrics.peersAskedForHeldDocuments,
		metrics.heldDocumentsAnsweringPeersRatio,
		metrics.unheldQueryWordsRatio,
		metrics.joinAskedMetadataForRatio,
		metrics.metadataThatCameBackRatio,
		metrics.wordJoinedSearchDurationSeconds,
	)

	return metrics
}

func bucketsFromNoneTo(ceiling float64, buckets int) []float64 {
	return append(
		[]float64{0},
		prometheusclient.ExponentialBucketsRange(1, ceiling, buckets)...,
	)
}

func (m *WordJoinedSearchMetrics) WordJoinedSearchPerformed(
	_ context.Context,
	search wordjoined.PerformedWordJoinedSearch,
) {
	m.peersAskedForHeldDocuments.Observe(float64(search.AmountOfPeersAskedForHeldDocuments))
	m.wordJoinedSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
	m.countJoin(search)
	m.observeHeldDocumentsAnsweringPeersRatio(search)
	m.observeQueryRatios(search)
}

func (m *WordJoinedSearchMetrics) countJoin(search wordjoined.PerformedWordJoinedSearch) {
	if search.AmountOfJoinedDocuments == 0 {
		m.wordJoinedSearches.WithLabelValues(joinFoundNoDocument).Inc()

		return
	}
	m.wordJoinedSearches.WithLabelValues(joinFoundDocuments).Inc()
	m.joinAskedMetadataForRatio.Observe(
		float64(search.AmountOfDocumentsToAskMetadataFor) / float64(search.AmountOfJoinedDocuments),
	)
}

func (m *WordJoinedSearchMetrics) observeHeldDocumentsAnsweringPeersRatio(
	search wordjoined.PerformedWordJoinedSearch,
) {
	if search.AmountOfPeersAskedForHeldDocuments == 0 {
		return
	}
	m.heldDocumentsAnsweringPeersRatio.Observe(
		float64(search.AmountOfPeersThatNamedHeldDocuments) /
			float64(search.AmountOfPeersAskedForHeldDocuments),
	)
}

func (m *WordJoinedSearchMetrics) observeQueryRatios(
	search wordjoined.PerformedWordJoinedSearch,
) {
	if search.AmountOfQueryWords > 0 {
		m.unheldQueryWordsRatio.Observe(
			float64(search.AmountOfQueryWordsNoPeerHeld) / float64(search.AmountOfQueryWords),
		)
	}
	if search.AmountOfDocumentsToAskMetadataFor == 0 {
		return
	}
	m.metadataThatCameBackRatio.Observe(
		float64(search.AmountOfDocumentsThatCameBack) /
			float64(search.AmountOfDocumentsToAskMetadataFor),
	)
}
