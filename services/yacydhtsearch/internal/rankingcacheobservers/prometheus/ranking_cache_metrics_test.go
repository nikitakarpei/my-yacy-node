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

func TestEveryOutcomeOfASearchIsPublishedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversprometheus.New(registry)
	query := searchquery.Query{Words: []string{"berlin"}}

	metrics.QueryAnsweredFromCache(t.Context(), query, 12)
	metrics.QueryAnsweredFromCache(t.Context(), query, 12)
	metrics.QueryAnsweredByPeers(t.Context(), query, 12)
	metrics.QueryHoldsNoIndexedTerm(t.Context(), query)
	metrics.QueryReachedNoPeer(t.Context(), query)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_searches_total{outcome="answered from cache"} 2`,
		`yacydhtsearch_searches_total{outcome="answered by peers"} 1`,
		`yacydhtsearch_searches_total{outcome="no indexed term"} 1`,
		`yacydhtsearch_searches_total{outcome="no peer reached"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASearchThatCameBackWithNoItemIsCountedApartFromOneThatHeldItems(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := rankingcacheobserversprometheus.New(registry)
	query := searchquery.Query{Words: []string{"berlin"}}

	metrics.QueryAnsweredByPeers(t.Context(), query, 0)
	metrics.QueryAnsweredFromCache(t.Context(), query, 0)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_searches_total{outcome="no item from peers"} 1`,
		`yacydhtsearch_searches_total{outcome="no item from cache"} 1`,
		`yacydhtsearch_searches_total{outcome="answered by peers"} 0`,
		`yacydhtsearch_searches_total{outcome="answered from cache"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
