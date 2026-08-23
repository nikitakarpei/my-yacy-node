package markdownstoremetrics_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownstoremetrics"
)

func TestMarkdownStoreMetricsRecordsAndExposesCounters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := markdownstoremetrics.New(registry)

	metrics.ScrapeRequestReceived()
	metrics.PageStored()
	metrics.StoreFailed()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpusmarkdown_scrape_requests_received_total 1",
		"corpusmarkdown_pages_stored_total 1",
		"corpusmarkdown_store_failures_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
