package rendermetrics_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendermetrics"
)

func TestRenderMetricsRecordsAndExposesCounters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := rendermetrics.New(registry)

	metrics.RenderSucceeded()
	metrics.RenderFailed("too_large")
	metrics.RenderWaited()
	metrics.RenderObserved(250 * time.Millisecond)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"renderproxy_renders_succeeded_total 1",
		`renderproxy_renders_failed_total{reason="too_large"} 1`,
		"renderproxy_render_waits_total 1",
		"renderproxy_render_duration_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
