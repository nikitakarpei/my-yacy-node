package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	dhtdistanceobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/dhtdistanceobservers/prometheus"
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

func TestEachPickedPeerPublishesItsDistanceFromTheTermItAnswersFor(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := dhtdistanceobserversprometheus.New(registry)

	metrics.PeersSelected(t.Context(), []float64{0.001, 0.5, 0.75})

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_selection_ring_fraction_count 3") ||
		!strings.Contains(body, "yacydhtsearch_selection_ring_fraction_sum 1.251") {
		t.Fatalf("metrics do not carry the ring fractions:\n%s", body)
	}
}
