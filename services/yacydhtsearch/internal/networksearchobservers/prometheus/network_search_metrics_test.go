package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	networksearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const queryBudget = 5 * time.Second

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestOneQueryPublishesThePeersItReachedAndWhatItCost(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskablePeers:   8,
		AmountOfItemsInRanking: 20,
		TimeSpent:              250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_network_search_askable_peers_sum 8",
		"yacydhtsearch_network_search_items_ranked_sum 20",
		"yacydhtsearch_network_search_duration_seconds_count 1",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAQueryOverTheBudgetIsCountedApartFromOneInsideIt(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskablePeers: 1, AmountOfItemsInRanking: 1,
		TimeSpent: queryBudget - time.Millisecond,
	})
	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskablePeers: 1, AmountOfItemsInRanking: 1,
		TimeSpent: queryBudget + time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_network_search_duration_seconds_bucket{le="5"} 1`,
		`yacydhtsearch_network_search_duration_seconds_bucket{le="6.25"} 2`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryOutcomeOfANetworkSearchIsPublishedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)
	query := searchquery.Query{Words: []string{"berlin"}}

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{})
	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{})
	metrics.QueryHoldsNoIndexedWord(t.Context(), query)
	metrics.QueryReachedNoPeer(t.Context(), query)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_network_searches_total{outcome="peers asked"} 2`,
		`yacydhtsearch_network_searches_total{outcome="no indexed word"} 1`,
		`yacydhtsearch_network_searches_total{outcome="no peer to ask"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
