package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
)

const labelEndedBy = "ended_by"

type urlMetadataAskMetrics struct {
	urlMetadataLookupsPerEndReason                  map[urlmetadataasks.EndReason]prometheusclient.Counter
	joinedDocumentsDroppedBeforeMetadataLookupRatio prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio           prometheusclient.Histogram
	urlMetadataAskDocuments                         prometheusclient.Histogram
}

func urlMetadataAskMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) urlMetadataAskMetrics {
	urlMetadataLookups := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_url_metadata_lookups_total",
		Help: "URL metadata lookups that asked at least one peer, by what ended them.",
	}, []string{labelEndedBy})
	metrics := urlMetadataAskMetrics{
		//exhaustive:enforce
		urlMetadataLookupsPerEndReason: map[urlmetadataasks.EndReason]prometheusclient.Counter{
			urlmetadataasks.EndedByCoverage: urlMetadataLookups.WithLabelValues(
				string(urlmetadataasks.EndedByCoverage),
			),
			urlmetadataasks.EndedByEveryAskSettled: urlMetadataLookups.WithLabelValues(
				string(urlmetadataasks.EndedByEveryAskSettled),
			),
			urlmetadataasks.EndedByCutoff: urlMetadataLookups.WithLabelValues(
				string(urlmetadataasks.EndedByCutoff),
			),
		},
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

func (m urlMetadataAskMetrics) observeURLMetadataAsks(
	spread wordjoined.PerformedWordJoinedSpread,
) {
	urlMetadataAsks := spread.URLMetadataAsks
	amountOfJoinedDocumentsWithoutMetadata := spread.AmountOfJoinedDocuments -
		spread.AmountOfJoinedDocumentsWithMetadata
	if amountOfJoinedDocumentsWithoutMetadata > 0 {
		m.joinedDocumentsDroppedBeforeMetadataLookupRatio.Observe(
			float64(amountOfJoinedDocumentsWithoutMetadata-urlMetadataAsks.AmountOfAskedDocuments) /
				float64(amountOfJoinedDocumentsWithoutMetadata),
		)
	}
	if urlMetadataAsks.AmountOfAskedDocuments == 0 {
		return
	}
	m.urlMetadataLookupsPerEndReason[urlMetadataAsks.EndReason].Inc()
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(
			urlMetadataAsks.AmountOfAskedDocuments-urlMetadataAsks.AmountOfAskedDocumentsWithMetadata,
		) /
			float64(urlMetadataAsks.AmountOfAskedDocuments),
	)
	for _, amountOfDocuments := range urlMetadataAsks.AmountOfDocumentsPerAsk {
		m.urlMetadataAskDocuments.Observe(float64(amountOfDocuments))
	}
}
