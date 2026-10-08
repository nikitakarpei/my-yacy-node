// Package prometheus reports how the URL metadata lookup of a word joined spread
// performed, as metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/ratiobuckets"
)

const labelEndedBy = "ended_by"

type URLMetadataLookupMetrics struct {
	lookupsPerEndReason                   map[urlmetadataasks.EndReason]prometheusclient.Counter
	documentsNotAskedRatio                prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio prometheusclient.Histogram
	askDocuments                          prometheusclient.Histogram
	timeToFirstAskSeconds                 prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	queryBudget time.Duration,
) *URLMetadataLookupMetrics {
	lookups := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_url_metadata_lookups_total",
		Help: "URL metadata lookups that asked at least one peer, by what ended them.",
	}, []string{labelEndedBy})
	metrics := &URLMetadataLookupMetrics{
		//exhaustive:enforce
		lookupsPerEndReason: map[urlmetadataasks.EndReason]prometheusclient.Counter{
			urlmetadataasks.EndedByCoverage: lookups.WithLabelValues(
				string(urlmetadataasks.EndedByCoverage),
			),
			urlmetadataasks.EndedByEveryAskSettled: lookups.WithLabelValues(
				string(urlmetadataasks.EndedByEveryAskSettled),
			),
			urlmetadataasks.EndedByCutoff: lookups.WithLabelValues(
				string(urlmetadataasks.EndedByCutoff),
			),
		},
		documentsNotAskedRatio: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_url_metadata_lookup_documents_not_asked_ratio",
				Help: "Share of the documents a URL metadata lookup was handed but never asked " +
					"any peer about.",
				Buckets: ratiobuckets.Tenths(),
			},
		),
		lookedUpDocumentsWithoutMetadataRatio: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio",
				Help: "Share of the documents the spread looked metadata up for that no peer " +
					"sent metadata for.",
				Buckets: ratiobuckets.Tenths(),
			},
		),
		askDocuments: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_spread_url_metadata_ask_documents",
			Help:    "Documents one URL metadata ask names, per ask sent.",
			Buckets: []float64{1, 2, 5, 10, 25, 50, 100, 200, 400, 700, 1000},
		}),
		timeToFirstAskSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_spread_time_to_first_url_metadata_ask_seconds",
			Help:    "Time from the start of a URL metadata lookup to its first ask, in seconds.",
			Buckets: budgetbuckets.DurationBucketsFor(queryBudget),
		}),
	}
	registry.MustRegister(
		lookups,
		metrics.documentsNotAskedRatio,
		metrics.lookedUpDocumentsWithoutMetadataRatio,
		metrics.askDocuments,
		metrics.timeToFirstAskSeconds,
	)

	return metrics
}

func (m *URLMetadataLookupMetrics) URLMetadataLookupPerformed(
	_ context.Context,
	lookup urlmetadataasks.Performed,
) {
	m.observeDocumentsNotAsked(lookup)
	if lookup.AmountOfLookedUpDocuments == 0 {
		return
	}
	m.lookupsPerEndReason[lookup.EndReason].Inc()
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(lookup.AmountOfLookedUpDocuments-lookup.AmountOfLookedUpDocumentsWithMetadata) /
			float64(lookup.AmountOfLookedUpDocuments),
	)
	for _, amountOfDocuments := range lookup.AmountOfDocumentsPerAsk {
		m.askDocuments.Observe(float64(amountOfDocuments))
	}
	if timeToFirstAsk, asked := lookup.TimeToFirstAsk.Get(); asked {
		m.timeToFirstAskSeconds.Observe(timeToFirstAsk.Seconds())
	}
}

func (m *URLMetadataLookupMetrics) observeDocumentsNotAsked(lookup urlmetadataasks.Performed) {
	amountOfDocumentsHanded := lookup.AmountOfLookedUpDocuments + lookup.AmountOfDocumentsNotAsked
	if amountOfDocumentsHanded == 0 {
		return
	}
	m.documentsNotAskedRatio.Observe(
		float64(lookup.AmountOfDocumentsNotAsked) / float64(amountOfDocumentsHanded),
	)
}
