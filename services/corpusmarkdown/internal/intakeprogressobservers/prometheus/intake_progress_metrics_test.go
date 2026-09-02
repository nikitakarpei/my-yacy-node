package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	intakeprogressmetrics "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakeprogressobservers/prometheus"
)

const pageURL = "https://example.org/a"

func TestIntakeProgressMetricsRecordsAndExposesCounters(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := intakeprogressmetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.PageOffered(ctx, page)
	metrics.MarkdownStored(ctx, page)
	metrics.NoDocumentExtracted(ctx, page, cause)
	metrics.NoMarkdownDerived(ctx, page)
	metrics.MarkdownNotStored(ctx, page, cause)
	metrics.IntakeReceiptNotSent(ctx, page, cause)

	req := httptest.NewRequestWithContext(ctx, "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpusmarkdown_pages_offered_total 1",
		`corpusmarkdown_offered_pages_disposed_total{disposal="stored"} 1`,
		`corpusmarkdown_offered_pages_disposed_total{disposal="no-document-extracted"} 1`,
		`corpusmarkdown_offered_pages_disposed_total{disposal="no-markdown-derived"} 1`,
		`corpusmarkdown_offered_pages_disposed_total{disposal="store-failed"} 1`,
		"corpusmarkdown_intake_receipt_failures_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
