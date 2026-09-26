package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	rankingcacheobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/rankingcacheobservers/jetstream/prometheus"
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

func TestAFailedHoldIsPublishedApartFromAFailedLookup(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversjetstreamprometheus.New(registry)
	query := searchquery.Query{Words: []string{"berlin"}}
	refused := errors.New("bucket refused")

	metrics.RankingLookupFailed(t.Context(), query, refused)
	metrics.RankingStoreFailed(t.Context(), query, refused)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_ranking_cache_failures_total{action="lookup"} 1`,
		`yacydhtsearch_ranking_cache_failures_total{action="store"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryRankingCacheFailureIsPublishedBeforeTheFirstFailure(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	rankingcacheobserversjetstreamprometheus.New(registry)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_ranking_cache_failures_total{action="lookup"} 0`,
		`yacydhtsearch_ranking_cache_failures_total{action="store"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
