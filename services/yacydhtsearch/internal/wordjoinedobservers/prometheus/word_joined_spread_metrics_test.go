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
	wordjoinedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordjoinedobservers/prometheus"
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
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		AmountOfQueryWords:                     4,
		AmountOfQueryWordsHeldByNoPeer:         1,
		AmountOfPeersAsked:                     8,
		AmountOfPeersThatAnswered:              6,
		TimeSpent:                              250 * time.Millisecond,
		AmountOfPeersHoldingAQueryWord:         4,
		AmountOfJoinedDocuments:                8,
		AmountOfJoinedDocumentsWithMetadata:    2,
		AmountOfMatchedDocumentsAcrossAnswers:  10,
		AmountOfMatchedDocumentsCountedByAPeer: 6,
		AmountOfDocumentsHeldInEachAnswer:      []int{16, 64},
		AmountOfDocumentsAskedMetadataFor:      4,
		AmountOfAskedDocumentsWithMetadata:     3,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 1`,
		"yacydhtsearch_word_joined_spread_join_asked_metadata_for_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_join_with_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_matched_documents_counted_by_a_peer_ratio_sum 0.6",
		"yacydhtsearch_word_joined_spread_documents_held_in_one_answer_sum 80",
		"yacydhtsearch_word_joined_spread_documents_held_in_one_answer_count 2",
		"yacydhtsearch_word_joined_spread_peers_asked_sum 8",
		"yacydhtsearch_word_joined_spread_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_asked_documents_with_metadata_ratio_sum 0.75",
		"yacydhtsearch_word_joined_spread_answering_peers_ratio_sum 0.75",
		"yacydhtsearch_word_joined_spread_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASpreadThatAskedNoPeerPublishesNoAnsweringRatio(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		AmountOfQueryWords: 2,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_answering_peers_ratio_count 0",
	) {
		t.Fatalf("metrics carry an answering ratio for a spread that asked no peer:\n%s", body)
	}
}

func TestASpreadNoDocumentHeldAllQueryWordsForIsCountedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		AmountOfQueryWords: 3,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="no document"} 1`,
		"yacydhtsearch_word_joined_spread_join_asked_metadata_for_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAJoinTheSpreadPutToNoPeerPublishesNoYield(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		AmountOfQueryWords:      2,
		AmountOfJoinedDocuments: 4,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_asked_documents_with_metadata_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share for a spread that asked no metadata:\n%s", body)
	}
}

func TestASpreadNoPeerMatchedADocumentInPublishesNoPostingShare(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		AmountOfQueryWords:      2,
		AmountOfJoinedDocuments: 4,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_matched_documents_counted_by_a_peer_ratio_count 0",
	) {
		t.Fatalf(
			"metrics carry a posting share for a spread no peer matched a document in:\n%s",
			body,
		)
	}
}
