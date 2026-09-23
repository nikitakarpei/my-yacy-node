package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

type urlMetadataRoundMetrics struct {
	joinedDocumentsDroppedBeforeMetadataLookupRatio prometheusclient.Histogram
	lookedUpDocumentsWithoutMetadataRatio           prometheusclient.Histogram
}

func urlMetadataRoundMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) urlMetadataRoundMetrics {
	metrics := urlMetadataRoundMetrics{
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
		metrics.joinedDocumentsDroppedBeforeMetadataLookupRatio,
		metrics.lookedUpDocumentsWithoutMetadataRatio,
	)

	return metrics
}

func (m urlMetadataRoundMetrics) observeURLMetadataRound(
	urlMetadataRound wordjoined.PerformedURLMetadataRound,
	crossCheckRound wordjoined.PerformedCrossCheckRound,
) {
	amountOfJoinedDocumentsWithoutMetadata := crossCheckRound.AmountOfJoinedDocuments -
		urlMetadataRound.AmountOfJoinedDocumentsWithMetadata
	if amountOfJoinedDocumentsWithoutMetadata > 0 {
		m.joinedDocumentsDroppedBeforeMetadataLookupRatio.Observe(
			float64(
				amountOfJoinedDocumentsWithoutMetadata-urlMetadataRound.AmountOfLookedUpDocuments,
			) /
				float64(
					amountOfJoinedDocumentsWithoutMetadata,
				),
		)
	}
	if urlMetadataRound.AmountOfLookedUpDocuments == 0 {
		return
	}
	m.lookedUpDocumentsWithoutMetadataRatio.Observe(
		float64(
			urlMetadataRound.AmountOfLookedUpDocuments-urlMetadataRound.AmountOfLookedUpDocumentsWithMetadata,
		) /
			float64(
				urlMetadataRound.AmountOfLookedUpDocuments,
			),
	)
}
