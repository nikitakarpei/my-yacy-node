package indexmetrics_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/indexmetrics"
)

func TestIndexMetricsRecordsAndExposesCounters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := indexmetrics.New(registry)

	metrics.PageReceived()
	metrics.PageIndexed()
	metrics.IndexFailed()
	metrics.IndexObserved(250 * time.Millisecond)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpustext_pages_received_total 1",
		"corpustext_pages_indexed_total 1",
		"corpustext_index_failures_total 1",
		"corpustext_index_duration_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
