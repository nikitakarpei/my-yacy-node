package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	pageintakemetrics "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintakeobservers/prometheus"
)

const pageURL = "https://example.org/a"

func TestPageIntakeMetricsRecordsAndExposesCounters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pageintakemetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.PageOffered(ctx, page)
	metrics.PageIndexed(ctx, page)
	metrics.NoDocumentExtracted(ctx, page, cause)
	metrics.NoReadableTextDerived(ctx, page)
	metrics.IndexFailed(ctx, page, cause)
	metrics.IndexWriteEnded(ctx, 250*time.Millisecond)

	req := httptest.NewRequestWithContext(ctx, "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpustext_pages_offered_total 1",
		`corpustext_offered_pages_disposed_total{disposal="indexed"} 1`,
		`corpustext_offered_pages_disposed_total{disposal="no-document-extracted"} 1`,
		`corpustext_offered_pages_disposed_total{disposal="no-readable-text-derived"} 1`,
		`corpustext_offered_pages_disposed_total{disposal="index-failed"} 1`,
		"corpustext_index_duration_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
