package visitmetrics_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitmetrics"
)

func TestMetricsRecordAndExpose(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := visitmetrics.New(registry)
	metrics.VisitReceived()
	metrics.VisitRejected()
	metrics.OrderPlaced()
	metrics.OrderUnplaced()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).
		ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil))

	body := recorder.Body.String()
	for _, want := range []string{
		"visitcrawl_visits_received_total 1",
		"visitcrawl_visits_rejected_total 1",
		"visitcrawl_orders_placed_total 1",
		"visitcrawl_orders_unplaced_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}
