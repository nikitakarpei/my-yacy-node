package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	documentamountsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/documentamounts/prometheus"
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

func assertPublished(t *testing.T, registry *prometheusclient.Registry, published ...string) {
	t.Helper()

	body := publishedBy(t, registry)
	for _, metric := range published {
		if !strings.Contains(body, metric) {
			t.Fatalf("metrics do not carry %q:\n%s", metric, body)
		}
	}
}

func TestEverySourceIsPublishedBeforeTheFirstAmounts(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	documentamountsprometheus.New(registry)

	assertPublished(t, registry,
		`yacydhtsearch_word_joined_spread_document_amount_sources_total{source="cache"} 0`,
		`yacydhtsearch_word_joined_spread_document_amount_sources_total{source="none"} 0`,
	)
}

func TestAmountsWithEveryQueryWordCachedCountForTheCache(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := documentamountsprometheus.New(registry)

	metrics.AmountsReadFromCache(t.Context(), documentamounts.PerformedFromCache{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 2, AllQueryWordsCached: true,
	})

	assertPublished(t, registry,
		`yacydhtsearch_word_joined_spread_document_amount_sources_total{source="cache"} 1`,
	)
}

func TestAmountsWithAQueryWordUncachedCountForNoSource(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := documentamountsprometheus.New(registry)

	metrics.AmountsReadFromCache(t.Context(), documentamounts.PerformedFromCache{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 1,
	})

	assertPublished(t, registry,
		`yacydhtsearch_word_joined_spread_document_amount_sources_total{source="none"} 1`,
		`yacydhtsearch_word_joined_spread_document_amount_sources_total{source="cache"} 0`,
	)
}
