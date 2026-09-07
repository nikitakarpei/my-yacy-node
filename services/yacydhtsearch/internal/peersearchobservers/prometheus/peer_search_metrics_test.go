package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peersearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

const (
	peerCallBudget    = 3 * time.Second
	peerResultCeiling = 10
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

func answeredItems(items int) []searchresult.Item {
	answered := make([]searchresult.Item, 0, items)
	for item := range items {
		answered = append(answered, searchresult.Item{
			Address: "https://" + strconv.Itoa(item) + ".example/",
		})
	}

	return answered
}

func TestEveryPeerCallIsCountedUnderItsOutcome(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peersearchobserversprometheus.New(registry, peerCallBudget, peerResultCeiling)

	metrics.PeerAnswered(t.Context(), "http://peer.example", answeredItems(3), time.Second)
	metrics.PeerRefused(
		t.Context(), "http://peer.example", http.StatusServiceUnavailable, time.Second,
	)
	metrics.PeerUnreachable(
		t.Context(),
		"http://peer.example",
		errors.New("no route"),
		peerCallBudget,
	)
	metrics.PeerAnswerUnreadable(
		t.Context(),
		"http://peer.example",
		errors.New("bad row"),
		time.Second,
	)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_calls_total{outcome="answered"} 1`,
		`yacydhtsearch_peer_calls_total{outcome="refused"} 1`,
		`yacydhtsearch_peer_calls_total{outcome="unreachable"} 1`,
		`yacydhtsearch_peer_calls_total{outcome="unreadable"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{outcome="answered"} 1`,
		`yacydhtsearch_peer_call_duration_seconds_sum{outcome="unreachable"} 3`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestAnAnswerPublishesHowManyItemsItCarried(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peersearchobserversprometheus.New(registry, peerCallBudget, peerResultCeiling)

	metrics.PeerAnswered(t.Context(), "http://peer.example", answeredItems(3), time.Second)
	metrics.PeerAnswered(
		t.Context(), "http://peer.example", answeredItems(peerResultCeiling), time.Second,
	)

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_peer_answer_items_count 2",
		"yacydhtsearch_peer_answer_items_sum 13",
		`yacydhtsearch_peer_answer_items_bucket{le="3"} 1`,
		`yacydhtsearch_peer_answer_items_bucket{le="10"} 2`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
