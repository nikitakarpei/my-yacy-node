package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	peercallobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallobservers/prometheus"
)

const queryBudget = 3 * time.Second

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEveryPeerCallIsCountedUnderItsOutcome(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	metrics.PeerAnsweredSearchDocuments(t.Context(), "http://peer.example", 0, 3, time.Second)
	metrics.PeerRefused(
		t.Context(),
		"http://peer.example",
		peerasks.SearchDocuments,
		http.StatusServiceUnavailable,
		time.Second,
	)
	metrics.PeerUnreachable(
		t.Context(),
		"http://peer.example",
		peerasks.SearchDocuments,
		errors.New("no route"),
		queryBudget,
	)
	metrics.PeerAnswerUnreadable(
		t.Context(),
		"http://peer.example",
		peerasks.SearchDocuments,
		errors.New("bad row"),
		time.Second,
	)
	metrics.PeerCallCancelled(
		t.Context(),
		"http://peer.example",
		peerasks.SearchDocuments,
		time.Second,
	)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="answered"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="refused"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="unreachable"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="unreadable"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="cancelled"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="url metadata",outcome="cancelled"} 0`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="search documents",outcome="cancelled"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="search documents",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="search documents",outcome="unreachable"} 3`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAPeerCallThatBroughtNothingIsCountedApartFromOneThatBroughtSomething(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	metrics.PeerAnsweredSearchDocuments(t.Context(), "http://peer.example", 0, 0, time.Second)
	metrics.PeerAnsweredURLMetadata(t.Context(), "http://peer.example", 0, time.Second)
	metrics.PeerAnsweredSearchDocuments(t.Context(), "http://peer.example", 0, 3, time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="answered nothing"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="url metadata",outcome="answered nothing"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="answered"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAnAnsweredMetadataCallIsCountedUnderWhatItAskedFor(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	metrics.PeerAnsweredURLMetadata(t.Context(), "http://peer.example", 4, time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="url metadata",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="url metadata",outcome="answered"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="answered"} 0`,
		`yacydhtsearch_peer_call_duration_seconds_count{asked_for="search documents",outcome="answered"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAnAnsweredWordIsCountedUnderWhatItAskedFor(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	metrics.PeerAnsweredSearchDocuments(t.Context(), "http://peer.example", 7, 0, time.Second)
	metrics.PeerAnsweredURLMetadata(t.Context(), "http://peer.example", 2, time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="search documents",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="search documents",outcome="answered"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="url metadata",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="url metadata",outcome="answered"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestPeerCallsWaitingForASlotAreCountedWhileTheyWait(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	if !strings.Contains(
		publishedBy(t, registry), "yacydhtsearch_peer_calls_waiting_for_a_slot 0",
	) {
		t.Fatalf("metrics do not start the waiting calls at zero:\n%s", publishedBy(t, registry))
	}

	metrics.PeerCallWaitsForASlot(t.Context(), "http://peer.example", peerasks.SearchDocuments)
	metrics.PeerCallWaitsForASlot(t.Context(), "http://peer.example", peerasks.URLMetadata)
	metrics.PeerCallTookASlot(
		t.Context(), "http://peer.example", peerasks.SearchDocuments, time.Second,
	)

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_peer_calls_waiting_for_a_slot 1") {
		t.Fatalf("metrics do not carry the one call still waiting for a slot:\n%s", body)
	}
}
