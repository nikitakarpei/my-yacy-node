package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	presenceaccrualobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrualobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func TestThePresenceAPeerEarnedIsPublishedWithTheFirstAnswersAndTheObservedPeers(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := presenceaccrualobserversprometheus.New(registry)
	peer := yacymodel.WordHash("berlin")

	metrics.PeerAnsweredForTheFirstTime(t.Context(), peer, "http://10.0.0.1:8090")
	metrics.PeerEarnedPresence(t.Context(), peer, time.Hour)
	metrics.PeersObserved(t.Context(), 7)

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_peer_presence_first_answers_total 1",
		"yacydhtsearch_peer_presence_earned_seconds_sum 3600",
		"yacydhtsearch_peer_presence_observed_peers 7",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
