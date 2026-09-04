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

	rendercapacityobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendercapacityobservers/prometheus"
)

func TestRenderCapacityMetricsRecordCapacityFacts(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := rendercapacityobserversprometheus.New(registry)
	metrics.RenderWaitedForCapacity(context.Background(), "https://example.com", time.Second)
	metrics.RenderEndedWhileWaitingForCapacity(
		context.Background(), "https://example.com", time.Second, errors.New("cancelled"),
	)
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder, httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil),
	)
	for _, expected := range []string{
		"renderproxy_render_capacity_wait_seconds_count 1",
		"renderproxy_renders_ended_while_waiting_for_capacity_total 1",
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Fatalf("metrics output does not contain %q:\n%s", expected, recorder.Body.String())
		}
	}
}
