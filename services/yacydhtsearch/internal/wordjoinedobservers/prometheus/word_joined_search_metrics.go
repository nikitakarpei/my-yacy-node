// Package prometheus reports how many peers the first round of a word joined
// search asked and how many answered, how much of the query the network held,
// how often no document held all query words, how many documents the peers hold
// for a query word, how much of the join the peers already reported as an item
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
	peerBucketCeiling             = 128.0
	peerBuckets                   = 8
	documentsPerWordBucketCeiling = 1048576.0
	documentsPerWordBuckets       = 11
	durationBuckets               = 12
	budgetShare                   = 1024
	ratioBuckets                  = 11
	ratioStep                     = 0.1
	labelJoin                     = "join"
	joinFoundNoDocument           = "no document"
	joinFoundDocuments            = "documents"
)

type WordJoinedSearchMetrics struct {
	wordJoinedSearches               *prometheusclient.CounterVec
	peersAskedForHeldDocuments       prometheusclient.Histogram
	heldDocumentsAnsweringPeersRatio prometheusclient.Histogram
	unheldQueryWordsRatio            prometheusclient.Histogram
	joinAlreadyReportedRatio         prometheusclient.Histogram
	reportedItemsWithAPostingRatio   prometheusclient.Histogram
	documentsAPeerHoldsPerQueryWord  prometheusclient.Histogram
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
		heldDocumentsAnsweringPeersRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_held_documents_answering_peers_ratio",
			"Share of the peers asked which documents they hold that answered.",
		),
		unheldQueryWordsRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_unheld_query_words_ratio",
			"Share of query words that no asked peer held a document for.",
		),
		joinAlreadyReportedRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_join_already_reported_ratio",
			"Share of the joined documents that a peer already reported as an item with "+
				"the documents it holds.",
		),
		reportedItemsWithAPostingRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_reported_items_with_a_posting_ratio",
			"Share of the items the peers reported with the documents they hold that "+
				"carried a posting.",
		),
		documentsAPeerHoldsPerQueryWord: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_search_documents_a_peer_holds_per_query_word",
				Help: "Documents one peer reported holding for one query word.",
				Buckets: bucketsFromNoneTo(
					documentsPerWordBucketCeiling, documentsPerWordBuckets,
				),
			},
		),
		joinAskedMetadataForRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_join_asked_metadata_for_ratio",
			"Share of the joined documents the search asked the peers metadata for.",
		),
		metadataThatCameBackRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_search_metadata_that_came_back_ratio",
			"Share of the documents the search asked metadata for that came back as an item.",
		),
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
		metrics.joinAlreadyReportedRatio,
		metrics.reportedItemsWithAPostingRatio,
		metrics.documentsAPeerHoldsPerQueryWord,
		metrics.joinAskedMetadataForRatio,
		metrics.metadataThatCameBackRatio,
		metrics.wordJoinedSearchDurationSeconds,
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

func (m *WordJoinedSearchMetrics) WordJoinedSearchPerformed(
	_ context.Context,
	search wordjoined.PerformedWordJoinedSearch,
) {
	m.peersAskedForHeldDocuments.Observe(float64(search.AmountOfPeersAskedForHeldDocuments))
	m.wordJoinedSearchDurationSeconds.Observe(search.TimeSpent.Seconds())
	m.countJoin(search)
	m.observeHeldDocumentsAnsweringPeersRatio(search)
	m.observeQueryRatios(search)
	m.observeWhatThePeersReported(search)
}

func (m *WordJoinedSearchMetrics) observeWhatThePeersReported(
	search wordjoined.PerformedWordJoinedSearch,
) {
	for _, documentsHeld := range search.DocumentsEachPeerHoldsForAQueryWord {
		m.documentsAPeerHoldsPerQueryWord.Observe(float64(documentsHeld))
	}
	if search.AmountOfReportedItems == 0 {
		return
	}
	m.reportedItemsWithAPostingRatio.Observe(
		float64(search.AmountOfReportedItemsWithAPosting) / float64(search.AmountOfReportedItems),
	)
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
	m.joinAlreadyReportedRatio.Observe(
		float64(search.AmountOfJoinedDocumentsAlreadyReported) /
			float64(search.AmountOfJoinedDocuments),
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
