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
		MatchedAndHeldDocumentsRound: wordjoined.PerformedMatchedAndHeldDocumentsRound{
			AmountOfQueryWords:                                    4,
			AmountOfQueryWordsHeldByNoPeer:                        1,
			AmountOfFullyListedQueryWords:                         2,
			AmountOfPeersAskedForMatchedAndHeldDocuments:          8,
			AmountOfPeersThatAnsweredMatchedAndHeldDocuments:      6,
			AmountOfPeersThatListedADocument:                      4,
			AmountOfDocumentsListedByThePeersOfTheRarestQueryWord: 12,
		},
		CrossCheckedDocumentsRound: wordjoined.PerformedCrossCheckedDocumentsRound{
			AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling:      3,
			AmountOfPeersAskedForCrossCheckedDocuments:                4,
			AmountOfPeersThatAnsweredCrossCheckedDocuments:            2,
			AmountOfJoinedDocumentsBeforeTheCrossCheckedDocumentsAsks: 4,
			AmountOfJoinedDocuments:                                   10,
		},
		URLMetadataRound: wordjoined.PerformedURLMetadataRound{
			AmountOfJoinedDocumentsWithMetadata: 2,
			AmountOfDocumentsAskedMetadataFor:   4,
			AmountOfAskedDocumentsWithMetadata:  3,
		},
		TimeSpent: 250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 1`,
		"yacydhtsearch_word_joined_spread_fully_listed_query_words_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_answering_cross_checked_documents_peers_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio_sum 0.6",
		"yacydhtsearch_word_joined_spread_documents_listed_for_the_rarest_query_word_that_missed_a_cross_check_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_peers_asked_for_matched_and_held_documents_sum 8",
		"yacydhtsearch_word_joined_spread_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_answering_matched_and_held_documents_peers_ratio_sum 0.75",
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
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(2))

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_answering_matched_and_held_documents_peers_ratio_count 0",
	) {
		t.Fatalf("metrics carry an answering ratio for a spread that asked no peer:\n%s", body)
	}
}

func spreadOfQueryWords(amountOfQueryWords int) wordjoined.PerformedWordJoinedSpread {
	return wordjoined.PerformedWordJoinedSpread{
		MatchedAndHeldDocumentsRound: wordjoined.PerformedMatchedAndHeldDocumentsRound{
			AmountOfQueryWords: amountOfQueryWords,
		},
	}
}

func TestASpreadThatCrossCheckedNoDocumentPublishesNoShareOfCrossChecking(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadJoiningDocuments(3))

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_word_joined_spread_answering_cross_checked_documents_peers_ratio_count 0",
		"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf(
				"metrics carry %q for a spread that cross-checked no document:\n%s",
				published,
				body,
			)
		}
	}
}

func TestASpreadNoDocumentHeldAllQueryWordsForIsCountedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(3))

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="no document"} 1`,
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
	spread.URLMetadataRound.AmountOfJoinedDocumentsWithMetadata = 4
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
	spread.CrossCheckedDocumentsRound.AmountOfJoinedDocuments = amountOfJoinedDocuments

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

func TestASpreadWhoseRarestQueryWordPeersListedNoDocumentPublishesNoShareThatMissedACrossCheck(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(2))

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_documents_listed_for_the_rarest_query_word_that_missed_a_cross_check_ratio_count 0",
	) {
		t.Fatalf(
			"metrics carry a share that missed a cross-check for a spread that listed no document:\n%s",
			body,
		)
	}
}
