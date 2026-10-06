package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	urlmetadataasksprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/urlmetadataasks/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func publishedAfter(t *testing.T, lookups ...urlmetadataasks.Performed) string {
	t.Helper()

	registry := prometheusclient.NewRegistry()
	metrics := urlmetadataasksprometheus.New(registry, 5*time.Second)
	for _, lookup := range lookups {
		metrics.URLMetadataLookupPerformed(t.Context(), lookup)
	}
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func assertCarries(t *testing.T, body string, published ...string) {
	t.Helper()

	for _, metric := range published {
		if !strings.Contains(body, metric) {
			t.Fatalf("metrics do not carry %q:\n%s", metric, body)
		}
	}
}

func TestOneLookupPublishesWhatItLookedUp(t *testing.T) {
	t.Parallel()

	body := publishedAfter(t, urlmetadataasks.Performed{
		AmountOfLookedUpDocuments:             4,
		AmountOfLookedUpDocumentsWithMetadata: 3,
		AmountOfDocumentsNotAsked:             4,
		EndReason:                             urlmetadataasks.EndedByCoverage,
		TimeToFirstAsk:                        yacymodel.Some(100 * time.Millisecond),
	})

	assertCarries(
		t,
		body,
		"yacydhtsearch_word_joined_spread_url_metadata_lookup_documents_not_asked_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_time_to_first_url_metadata_ask_seconds_sum 0.1",
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="coverage"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
	)
}

func TestEveryEndOfALookupIsPublishedBeforeTheFirstLookup(t *testing.T) {
	t.Parallel()

	assertCarries(
		t,
		publishedAfter(t),
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="coverage"} 0`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="cut off"} 0`,
	)
}

func TestALookupThatWasCutOffIsCountedByItsCutoff(t *testing.T) {
	t.Parallel()

	body := publishedAfter(t, urlmetadataasks.Performed{
		AmountOfLookedUpDocuments:             4,
		AmountOfLookedUpDocumentsWithMetadata: 3,
		EndReason:                             urlmetadataasks.EndedByCutoff,
		AmountOfDocumentsCutOff:               1,
	})

	assertCarries(t, body,
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="cut off"} 1`,
	)
}

func TestEachSentURLMetadataAskIsCountedByItsSize(t *testing.T) {
	t.Parallel()

	body := publishedAfter(t, urlmetadataasks.Performed{
		AmountOfLookedUpDocuments:             425,
		AmountOfLookedUpDocumentsWithMetadata: 425,
		EndReason:                             urlmetadataasks.EndedByCoverage,
		AmountOfDocumentsPerAsk:               []int{25, 400},
	})

	assertCarries(t, body,
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="25"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="200"} 1`,
		`yacydhtsearch_word_joined_spread_url_metadata_ask_documents_bucket{le="400"} 2`,
		"yacydhtsearch_word_joined_spread_url_metadata_ask_documents_count 2",
	)
}

func TestALookupHandedNoDocumentPublishesNoShare(t *testing.T) {
	t.Parallel()

	body := publishedAfter(t, urlmetadataasks.Performed{
		EndReason: urlmetadataasks.EndedByEveryAskSettled,
	})

	assertCarries(
		t,
		body,
		"yacydhtsearch_word_joined_spread_url_metadata_lookup_documents_not_asked_ratio_count 0",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_count 0",
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
	)
}

func TestALookupThatAskedNoPeerAboutItsDocumentsPublishesOnlyTheShareNotAsked(t *testing.T) {
	t.Parallel()

	body := publishedAfter(t, urlmetadataasks.Performed{
		AmountOfDocumentsNotAsked: 2,
		EndReason:                 urlmetadataasks.EndedByEveryAskSettled,
	})

	assertCarries(
		t,
		body,
		"yacydhtsearch_word_joined_spread_url_metadata_lookup_documents_not_asked_ratio_sum 1",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_count 0",
		`yacydhtsearch_word_joined_spread_url_metadata_lookups_total{ended_by="every ask settled"} 0`,
	)
}
