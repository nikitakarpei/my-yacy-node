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
		AmountOfAskablePeers:            8,
		AmountOfItemsInRanking:          20,
		AmountOfRankedItemsOfTheOnePeer: 5,
		TimeSpent:                       250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_network_search_askable_peers_sum 8",
		"yacydhtsearch_network_search_items_ranked_sum 20",
		"yacydhtsearch_network_search_ranking_top_peer_share_sum 0.25",
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

func TestOneQueryPublishesHowMuchOfTheRankingAPeerCounted(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskablePeers:              4,
		AmountOfItemsInRanking:            10,
		AmountOfRankedItemsCountedByAPeer: 6,
		TimeSpent:                         time.Second,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_network_search_ranked_items_with_a_posting_ratio_sum 0.6",
		"yacydhtsearch_network_search_ranked_items_with_a_posting_ratio_count 1",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASearchThatRankedNoItemObservesNoRankingShare(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskablePeers: 4, TimeSpent: time.Second,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_network_search_ranking_top_peer_share_count 0",
		"yacydhtsearch_network_search_ranked_items_with_a_posting_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics carry %q for a search that ranked no item:\n%s", published, body)
		}
	}
}
