package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	rankingcacheobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
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

func TestHitsAndMissesArePublishedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversprometheus.New(registry)
	query := searchquery.Query{Words: []string{"berlin"}}

	metrics.QueryAnsweredFromCache(t.Context(), query)
	metrics.QueryAnsweredFromCache(t.Context(), query)
	metrics.QueryMissedCache(t.Context(), query)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_ranking_cache_lookups_total{result="hit"} 2`,
		`yacydhtsearch_ranking_cache_lookups_total{result="miss"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
