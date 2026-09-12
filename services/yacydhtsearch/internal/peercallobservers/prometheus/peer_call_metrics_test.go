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

	metrics.PeerAnsweredMatchedItems(t.Context(), "http://peer.example", 3, time.Second)
	metrics.PeerRefused(
		t.Context(),
		"http://peer.example",
		peerasks.MatchedItems,
		http.StatusServiceUnavailable,
		time.Second,
	)
	metrics.PeerUnreachable(
		t.Context(),
		"http://peer.example",
		peerasks.MatchedItems,
		errors.New("no route"),
		queryBudget,
	)
	metrics.PeerAnswerUnreadable(
		t.Context(),
		"http://peer.example",
		peerasks.MatchedItems,
		errors.New("bad row"),
		time.Second,
	)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="answered"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="refused"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="unreachable"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="unreadable"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="matched items",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="matched items",outcome="unreachable"} 3`,
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

	metrics.PeerAnsweredMatchedItems(t.Context(), "http://peer.example", 0, time.Second)
	metrics.PeerAnsweredHeldDocuments(t.Context(), "http://peer.example", 0, time.Second)
	metrics.PeerAnsweredURLMetadata(t.Context(), "http://peer.example", 0, time.Second)
	metrics.PeerAnsweredMatchedItems(t.Context(), "http://peer.example", 3, time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="answered nothing"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="held documents",outcome="answered nothing"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="url metadata",outcome="answered nothing"} 1`,
		`yacydhtsearch_peer_calls_total{asked_for="matched items",outcome="answered"} 1`,
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
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
	if strings.Contains(body, `asked_for="matched items"`) {
		t.Fatalf("a metadata answer was counted as a peer call that asked for items:\n%s", body)
	}
}

func TestAnAnsweredWordIsCountedUnderWhatItAskedFor(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peercallobserversprometheus.New(registry, queryBudget)

	metrics.PeerAnsweredHeldDocuments(t.Context(), "http://peer.example", 7, time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{asked_for="held documents",outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{asked_for="held documents",outcome="answered"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
