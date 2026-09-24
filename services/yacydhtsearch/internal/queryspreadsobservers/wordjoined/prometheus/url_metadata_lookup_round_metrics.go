package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const labelEndedBy = "ended_by"

type urlMetadataLookupRoundMetrics struct {
	urlMetadataLookupsPerEnd                        map[wordjoined.URLMetadataLookupEnd]prometheusclient.Counter
	joinedDocumentsDroppedBeforeMetadataLookupRatio prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio           prometheusclient.Histogram
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
		urlMetadataLookupsPerEnd: map[wordjoined.URLMetadataLookupEnd]prometheusclient.Counter{
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
			"Share of the joined documents without metadata that the spread dropped before "+
				"looking their metadata up.",
		),
		lookedUpDocumentsWithoutMetadataRatio: ratioHistogramNamed(
			"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio",
			"Share of the documents the spread looked metadata up for that no peer sent "+
				"metadata for.",
		),
	}
	registry.MustRegister(
		urlMetadataLookups,
		metrics.joinedDocumentsDroppedBeforeMetadataLookupRatio,
		metrics.lookedUpDocumentsWithoutMetadataRatio,
	)

	return metrics
}

func (m urlMetadataLookupRoundMetrics) observeURLMetadataLookupRound(
	urlMetadataLookupRound wordjoined.PerformedURLMetadataLookupRound,
	amountOfJoinedDocuments int,
) {
	amountOfJoinedDocumentsWithoutMetadata := amountOfJoinedDocuments -
		urlMetadataLookupRound.AmountOfJoinedDocumentsWithMetadata
	if amountOfJoinedDocumentsWithoutMetadata > 0 {
		m.joinedDocumentsDroppedBeforeMetadataLookupRatio.Observe(
			float64(
				amountOfJoinedDocumentsWithoutMetadata-urlMetadataLookupRound.AmountOfLookedUpDocuments,
			) /
				float64(
					amountOfJoinedDocumentsWithoutMetadata,
				),
		)
	}
	if urlMetadataLookupRound.AmountOfLookedUpDocuments == 0 {
		return
	}
	m.urlMetadataLookupsPerEnd[urlMetadataLookupRound.End].Inc()
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(
			urlMetadataLookupRound.AmountOfLookedUpDocuments-urlMetadataLookupRound.AmountOfLookedUpDocumentsWithMetadata,
		) /
			float64(
				urlMetadataLookupRound.AmountOfLookedUpDocuments,
			),
	)
}
