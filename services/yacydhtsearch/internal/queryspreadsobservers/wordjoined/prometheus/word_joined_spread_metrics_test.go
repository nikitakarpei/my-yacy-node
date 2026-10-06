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
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	queryspreadsobserverswordjoinedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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
		DocumentAsks: documentasks.Performed{
			AmountOfPeersWithANonEmptyAbstract: 4,
		},
		AmountOfQueryWords:                  4,
		AmountOfQueryWordsHeldByNoPeer:      1,
		AmountOfDocumentsOfTheLeadingWord:   12,
		AmountOfJoinedDocuments:             10,
		AmountOfJoinedDocumentsWithMetadata: 2,
		URLMetadataAsks: urlmetadataasks.Performed{
			AmountOfLookedUpDocuments:             4,
			AmountOfLookedUpDocumentsWithMetadata: 3,
			EndReason:                             urlmetadataasks.EndedByCoverage,
			TimeToFirstAsk:                        yacymodel.Some(100 * time.Millisecond),
		},
		TimeSpent: 250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 1`,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_duration_seconds_sum 0.25",
		"yacydhtsearch_word_joined_spread_time_to_first_url_metadata_ask_seconds_sum 0.1",
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="coverage"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
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
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document"} 0`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="coverage"} 0`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="cut off"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestALookupThatWasCutOffIsCountedByItsCutoff(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadJoiningDocuments(4)
	spread.URLMetadataAsks = urlmetadataasks.Performed{
		AmountOfLookedUpDocuments:             4,
		AmountOfLookedUpDocumentsWithMetadata: 3,
		EndReason:                             urlmetadataasks.EndedByCutoff,
		AmountOfDocumentsCutOff:               1,
	}
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="cut off"} 1`,
	) {
		t.Fatalf("metrics do not count the lookup that was cut off:\n%s", body)
	}
}

func TestEachSentURLMetadataAskIsCountedByItsSize(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadJoiningDocuments(425)
	spread.URLMetadataAsks = urlmetadataasks.Performed{
		AmountOfLookedUpDocuments:             425,
		AmountOfLookedUpDocumentsWithMetadata: 425,
		EndReason:                             urlmetadataasks.EndedByCoverage,
		AmountOfDocumentsPerAsk:               []int{25, 400},
	}
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="25"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="200"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="400"} 2`,
		"yacydhtsearch_word_joined_spread_url_metadata_ask_documents_count 2",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func spreadOfQueryWords(amountOfQueryWords int) wordjoined.PerformedWordJoinedSpread {
	return wordjoined.PerformedWordJoinedSpread{AmountOfQueryWords: amountOfQueryWords}
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
	spread.AmountOfJoinedDocumentsWithMetadata = 4
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

func TestALookupThatAskedNoPeerIsNotCountedByWhatEndedIt(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadJoiningDocuments(4)
	spread.URLMetadataAsks.EndReason = urlmetadataasks.EndedByEveryAskSettled
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
	) {
		t.Fatalf("metrics count a lookup that asked no peer:\n%s", body)
	}
}
