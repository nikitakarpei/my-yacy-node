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
		AmountOfAskedPeers:                 8,
		AmountOfAnsweringPeers:             4,
		AmountOfPeersThatSentItems:         2,
		AmountOfItemsAcrossAnswers:         40,
		AmountOfRepeatedItemsAcrossAnswers: 20,
		AmountOfItemsInRanking:             20,
		TimeSpent:                          250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_network_search_peers_asked_sum 8",
		"yacydhtsearch_network_search_answering_peers_ratio_sum 0.5",
		"yacydhtsearch_network_search_peers_that_sent_items_ratio_sum 0.5",
		"yacydhtsearch_network_search_overlap_ratio_sum 0.5",
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
		AmountOfAskedPeers: 1, AmountOfAnsweringPeers: 1, AmountOfItemsInRanking: 1,
		TimeSpent: queryBudget - time.Millisecond,
	})
	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskedPeers: 1, AmountOfAnsweringPeers: 1, AmountOfItemsInRanking: 1,
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

func TestASearchThatNoPeerAnsweredObservesNoItemShare(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskedPeers: 4, AmountOfAnsweringPeers: 0, TimeSpent: time.Second,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(
		body, "yacydhtsearch_network_search_peers_that_sent_items_ratio_count 0",
	) {
		t.Fatalf("metrics carry an item share for a search that no peer answered:\n%s", body)
	}
}

func TestASearchThatCarriedNoItemObservesNoOverlap(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := networksearchobserversprometheus.New(registry, queryBudget)

	metrics.NetworkSearchPerformed(t.Context(), networksearch.PerformedNetworkSearch{
		AmountOfAskedPeers: 4, AmountOfAnsweringPeers: 0, TimeSpent: time.Second,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_network_search_overlap_ratio_count 0") {
		t.Fatalf("metrics carry an overlap for a search that had no item:\n%s", body)
	}
}
