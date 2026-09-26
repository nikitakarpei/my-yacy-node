package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const labelEndedBy = "ended_by"

type urlMetadataLookupRoundMetrics struct {
	urlMetadataLookupsPerEndReason                  map[wordjoined.URLMetadataLookupEndReason]prometheusclient.Counter
	joinedDocumentsDroppedBeforeMetadataLookupRatio prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio           prometheusclient.Histogram
	urlMetadataAskDocuments                         prometheusclient.Histogram
}

func urlMetadataLookupRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) urlMetadataLookupRoundMetrics {
	urlMetadataLookups := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_url_metadata_lookups_total",
		Help: "URL metadata lookups that asked at least one peer, by what ended them.",
	}, []string{labelEndedBy})
	metrics := urlMetadataLookupRoundMetrics{
		//exhaustive:enforce
		urlMetadataLookupsPerEndReason: map[wordjoined.URLMetadataLookupEndReason]prometheusclient.Counter{
			wordjoined.URLMetadataLookupEndedByCoverage: urlMetadataLookups.WithLabelValues(
				string(wordjoined.URLMetadataLookupEndedByCoverage),
			),
			wordjoined.URLMetadataLookupEndedByEveryAskSettled: urlMetadataLookups.WithLabelValues(
				string(wordjoined.URLMetadataLookupEndedByEveryAskSettled),
			),
			wordjoined.URLMetadataLookupEndedByCutoff: urlMetadataLookups.WithLabelValues(
				string(wordjoined.URLMetadataLookupEndedByCutoff),
			),
		},
		joinedDocumentsDroppedBeforeMetadataLookupRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio",
			"Share of the joined documents that the spread dropped before looking their "+
				"metadata up.",
		),
		lookedUpDocumentsWithoutMetadataRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio",
			"Share of the documents the spread looked metadata up for that no peer sent "+
				"metadata for.",
		),
		urlMetadataAskDocuments: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_word_joined_spread_url_metadata_ask_documents",
			Help:    "Documents one URL metadata ask names, per ask sent.",
			Buckets: []float64{25, 50, 100, 200, 400, 700, 1000},
		}),
	}
	registry.MustRegister(
		urlMetadataLookups,
		metrics.joinedDocumentsDroppedBeforeMetadataLookupRatio,
		metrics.lookedUpDocumentsWithoutMetadataRatio,
		metrics.urlMetadataAskDocuments,
	)

	return metrics
}

func (m urlMetadataLookupRoundMetrics) observeURLMetadataLookupRound(
	urlMetadataLookupRound wordjoined.PerformedURLMetadataLookupRound,
	amountOfJoinedDocuments int,
) {
	if amountOfJoinedDocuments > 0 {
		m.joinedDocumentsDroppedBeforeMetadataLookupRatio.Observe(
			float64(amountOfJoinedDocuments-urlMetadataLookupRound.AmountOfLookedUpDocuments) /
				float64(amountOfJoinedDocuments),
		)
	}
	if urlMetadataLookupRound.AmountOfLookedUpDocuments == 0 {
		return
	}
	m.urlMetadataLookupsPerEndReason[urlMetadataLookupRound.EndReason].Inc()
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(
			urlMetadataLookupRound.AmountOfLookedUpDocuments-urlMetadataLookupRound.AmountOfLookedUpDocumentsWithMetadata,
		) /
			float64(
				urlMetadataLookupRound.AmountOfLookedUpDocuments,
			),
	)
	for _, amountOfDocuments := range urlMetadataLookupRound.AmountOfDocumentsPerAsk {
		m.urlMetadataAskDocuments.Observe(float64(amountOfDocuments))
	}
}
