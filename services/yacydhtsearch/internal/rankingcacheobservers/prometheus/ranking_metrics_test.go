package prometheus_test

import (
	"errors"
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

func TestARankingPublishesWhereItCameFrom(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversprometheus.New(registry)
	query := searchquery.QueryFrom("berlin")

	metrics.NetworkRankingAnswered(t.Context(), query, 12)
	metrics.CachedRankingAnswered(t.Context(), query, 12)
	metrics.CachedRankingAnswered(t.Context(), query, 12)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_ranking_lookups_total{source="network"} 1`,
		`yacydhtsearch_ranking_lookups_total{source="cache"} 2`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAFailedHoldIsPublishedApartFromAFailedLookup(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversprometheus.New(registry)
	query := searchquery.QueryFrom("berlin")
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
