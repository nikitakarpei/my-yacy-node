package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
	renderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderobservers/prometheus"
)

func TestRenderMetricsRecordExplicitRenderFacts(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := renderobserversprometheus.New(registry)
	metrics.RenderSucceeded(t.Context(), "https://example.com", time.Second)
	metrics.RenderFailed(
		t.Context(),
		"https://example.com",
		250*time.Millisecond,
		rendergate.RenderFailurePageTooLarge,
		errors.New("too large"),
	)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, expected := range []string{
		`renderproxy_renders_processed_total{outcome="succeeded"} 1`,
		`renderproxy_renders_processed_total{outcome="page_too_large"} 1`,
		`renderproxy_render_duration_seconds_count 2`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output does not contain %q:\n%s", expected, body)
		}
	}
}
