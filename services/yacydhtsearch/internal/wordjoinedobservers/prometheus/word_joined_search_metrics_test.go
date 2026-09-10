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

func TestOneWordJoinedSearchPublishesWhatTheJoinFound(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSearchPerformed(t.Context(), wordjoined.PerformedWordJoinedSearch{
		AmountOfQueryWords:                     4,
		AmountOfQueryWordsNoPeerHeld:           1,
		AmountOfPeersAskedForHeldDocuments:     8,
		AmountOfPeersThatNamedHeldDocuments:    6,
		TimeSpent:                              250 * time.Millisecond,
		AmountOfPeersHoldingAQueryWord:         4,
		AmountOfJoinedDocuments:                8,
		AmountOfJoinedDocumentsAlreadyReported: 2,
		AmountOfReportedItems:                  10,
		AmountOfReportedItemsWithAPosting:      6,
		DocumentsEachPeerHoldsForAQueryWord:    []int{16, 64},
		AmountOfDocumentsToAskMetadataFor:      4,
		AmountOfDocumentsThatCameBack:          3,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_searches_total{join="documents"} 1`,
		"yacydhtsearch_word_joined_search_join_asked_metadata_for_ratio_sum 0.5",
		"yacydhtsearch_word_joined_search_join_already_reported_ratio_sum 0.25",
		"yacydhtsearch_word_joined_search_reported_items_with_a_posting_ratio_sum 0.6",
		"yacydhtsearch_word_joined_search_documents_a_peer_holds_per_query_word_sum 80",
		"yacydhtsearch_word_joined_search_documents_a_peer_holds_per_query_word_count 2",
		"yacydhtsearch_word_joined_search_peers_asked_for_held_documents_sum 8",
		"yacydhtsearch_word_joined_search_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_search_metadata_that_came_back_ratio_sum 0.75",
		"yacydhtsearch_word_joined_search_held_documents_answering_peers_ratio_sum 0.75",
		"yacydhtsearch_word_joined_search_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASearchThatAskedNoPeerPublishesNoAnsweringRatio(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSearchPerformed(t.Context(), wordjoined.PerformedWordJoinedSearch{
		AmountOfQueryWords: 2,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_search_held_documents_answering_peers_ratio_count 0",
	) {
		t.Fatalf("metrics carry an answering ratio for a search that asked no peer:\n%s", body)
	}
}

func TestASearchNoDocumentHeldAllQueryWordsForIsCountedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSearchPerformed(t.Context(), wordjoined.PerformedWordJoinedSearch{
		AmountOfQueryWords: 3,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_searches_total{join="no document"} 1`,
		"yacydhtsearch_word_joined_search_join_asked_metadata_for_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAJoinTheSearchPutToNoPeerPublishesNoYield(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSearchPerformed(t.Context(), wordjoined.PerformedWordJoinedSearch{
		AmountOfQueryWords:      2,
		AmountOfJoinedDocuments: 4,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_search_metadata_that_came_back_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share for a search that asked no metadata:\n%s", body)
	}
}

func TestASearchNoPeerReportedAnItemForPublishesNoPostingShare(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := wordjoinedobserversprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSearchPerformed(t.Context(), wordjoined.PerformedWordJoinedSearch{
		AmountOfQueryWords:      2,
		AmountOfJoinedDocuments: 4,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_search_reported_items_with_a_posting_ratio_count 0",
	) {
		t.Fatalf(
			"metrics carry a posting share for a search no peer reported an item for:\n%s",
			body,
		)
	}
}
