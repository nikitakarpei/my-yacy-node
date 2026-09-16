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

func TestThePresenceAPeerEarnedIsPublished(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := presenceaccrualobserversprometheus.New(registry)

	metrics.PeerEarnedPresence(t.Context(), yacymodel.WordHash("berlin"), time.Hour)

	body := publishedBy(t, registry)
	published := "yacydhtsearch_peer_presence_earned_seconds_sum 3600"
	if !strings.Contains(body, published) {
		t.Fatalf("metrics do not carry %q:\n%s", published, body)
	}
}
