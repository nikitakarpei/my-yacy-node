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
		AmountOfQueryWords:                  4,
		AmountOfQueryWordsHeldByNoPeer:      1,
		AmountOfJoinedDocuments:             10,
		AmountOfJoinedDocumentsWithMetadata: 2,
		TimeSpent:                           250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 1`,
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
		`yacydhtsearch_word_joined_spreads_total{join="documents"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document"} 0`,
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
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
