package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	documentasksprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentasks/prometheus"
)

func assertPublished(t *testing.T, registry *prometheusclient.Registry, published ...string) {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)
	body := recorder.Body.String()
	for _, metric := range published {
		if !strings.Contains(body, metric) {
			t.Fatalf("metrics do not carry %q:\n%s", metric, body)
		}
	}
}

func TestEveryKindOfOtherWordAskIsPublishedBeforeTheFirstAsk(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	documentasksprometheus.New(registry)

	assertPublished(
		t,
		registry,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="naming the documents to match"} 0`,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="skipped, no documents to match"} 0`,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="naming none, over the ceiling"} 0`,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="naming none, predicted over the ceiling"} 0`,
	)
}

func TestEachAskedPartitionIsCountedByItsKind(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := documentasksprometheus.New(registry)

	metrics.AskedPartitionFor(t.Context(), 0, documentasks.NamingTheDocumentsToMatch)
	metrics.AskedPartitionFor(t.Context(), 1, documentasks.Skipped)
	metrics.AskedPartitionFor(t.Context(), 2, documentasks.NamingTheDocumentsToMatch)
	metrics.AskedPartitionFor(t.Context(), 3, documentasks.OverTheCeiling)

	assertPublished(
		t,
		registry,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="naming the documents to match"} 2`,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="skipped, no documents to match"} 1`,
		`yacydhtsearch_word_joined_spread_other_word_ask_partitions_total{other_word_asks="naming none, over the ceiling"} 1`,
	)
}
