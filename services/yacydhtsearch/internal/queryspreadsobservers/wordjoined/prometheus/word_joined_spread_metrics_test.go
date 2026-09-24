package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	queryspreadsobserverswordjoinedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/prometheus"
)

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestOneWordJoinedSpreadPublishesWhatTheJoinFound(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		DiscoveryRound: wordjoined.PerformedDiscoveryRound{
			AmountOfQueryWords:                     4,
			AmountOfQueryWordsHeldByNoPeer:         1,
			AmountOfSampledQueryWords:              2,
			AmountOfPeersWithANonEmptyAbstract:     4,
			LeadingQueryWordChoice:                 wordjoined.RarestSampledQueryWord,
			AmountOfDocumentsOfTheLeadingQueryWord: 12,
			OtherWordsAskingPerPartition: []wordjoined.OtherWordsAsking{
				wordjoined.OtherWordsAskedForTheCandidates,
				wordjoined.OtherWordsNotAsked,
				wordjoined.OtherWordsAskedForTheCandidates,
				wordjoined.OtherWordsAskedBesideAPartialLeadingWord,
			},
		},
		AmountOfJoinedDocuments: 10,
		URLMetadataLookupRound: wordjoined.PerformedURLMetadataLookupRound{
			AmountOfJoinedDocumentsWithMetadata:   2,
			AmountOfLookedUpDocuments:             4,
			AmountOfLookedUpDocumentsWithMetadata: 3,
		},
		TimeSpent: 250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="rarest sampled word"} 1`,
		"yacydhtsearch_word_joined_spread_sampled_query_words_ratio_sum 0.5",
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="for the candidates"} 2`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="no candidates"} 1`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="whole, partial leading word"} 1`,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryKindOfWordJoinedSpreadIsPublishedBeforeTheFirstSpread(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="rarest sampled word"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest sampled word"} 0`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="for the candidates"} 0`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="no candidates"} 0`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="whole, candidates over the ceiling"} 0`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="whole, partial leading word"} 0`,
		`yacydhtsearch_word_joined_spread_partitions_total{other_words_asking="whole, no sample"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="rarest word, partial abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest word, partial abstracts"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func spreadOfQueryWords(amountOfQueryWords int) wordjoined.PerformedWordJoinedSpread {
	return wordjoined.PerformedWordJoinedSpread{
		DiscoveryRound: wordjoined.PerformedDiscoveryRound{
			AmountOfQueryWords:     amountOfQueryWords,
			LeadingQueryWordChoice: wordjoined.RarestQueryWordWithoutCompleteAbstracts,
		},
	}
}

func TestASpreadNoDocumentHeldAllQueryWordsForIsCountedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(3))

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest word, partial abstracts"} 1`,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASpreadWhoseDocumentsAllCarriedMetadataPublishesNoShareDroppedBeforeLookup(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadJoiningDocuments(4)
	spread.URLMetadataLookupRound.AmountOfJoinedDocumentsWithMetadata = 4
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share dropped before lookup for a spread missing none:\n%s", body)
	}
}

func spreadJoiningDocuments(amountOfJoinedDocuments int) wordjoined.PerformedWordJoinedSpread {
	spread := spreadOfQueryWords(2)
	spread.AmountOfJoinedDocuments = amountOfJoinedDocuments

	return spread
}

func TestASpreadThatLookedUpNoMetadataPublishesNoShareWithoutMetadata(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadJoiningDocuments(4))

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share for a spread that looked up no metadata:\n%s", body)
	}
}
