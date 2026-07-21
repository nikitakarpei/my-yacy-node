package markdownstoremetrics

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarkdownStoreMetricsRecordsAndExposesCounters(t *testing.T) {
	metrics := New()

	metrics.PageReceived()
	metrics.PageStored()
	metrics.StoreFailed()

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpusmarkdown_pages_received_total 1",
		"corpusmarkdown_pages_stored_total 1",
		"corpusmarkdown_store_failures_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
