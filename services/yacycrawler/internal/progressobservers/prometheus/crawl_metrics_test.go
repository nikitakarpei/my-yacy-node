package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
)

func TestMetricsRecordAndExpose(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := progressobserversprometheus.New(registry)
	metrics.OrderReceived()
	metrics.OrderAccepted()
	metrics.OrderReturned()
	metrics.PageFetched()
	metrics.ScrapeRequestPublished()
	metrics.PageDisposed("unsupported-media-type")
	metrics.AccessRefusalHonored()
	metrics.DeferralHonored()
	metrics.IndexingRefusalHonored()
	metrics.LinkDiscoveryRefusalHonored()
	metrics.FetchTook(250 * time.Millisecond)

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).
		ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil))

	body := recorder.Body.String()
	for _, want := range []string{
		"yacycrawler_orders_received_total 1",
		"yacycrawler_scrape_requests_published_total 1",
		"yacycrawler_pages_disposed_total",
		`yacycrawler_refusals_honored_total{demand="access-refusal"} 1`,
		`yacycrawler_refusals_honored_total{demand="deferral"} 1`,
		`yacycrawler_refusals_honored_total{demand="indexing-refusal"} 1`,
		`yacycrawler_refusals_honored_total{demand="link-discovery-refusal"} 1`,
		"yacycrawler_fetch_duration_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}
