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

func TestPageIntakeMetricsCountDisposalsRetriesAndIndexWrites(t *testing.T) {
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

	body := exposition(t, registry)
	for _, want := range []string{
		`corpustext_offered_pages_disposed_total{disposal="indexed"} 1`,
		`corpustext_offered_pages_disposed_total{disposal="no-document-extracted"} 1`,
		`corpustext_offered_pages_disposed_total{disposal="no-readable-text-derived"} 1`,
		`corpustext_pages_left_for_retry_total{cause="index-failed"} 1`,
		"corpustext_index_duration_seconds",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestPageIntakeMetricsKeepRetriedPagesOutOfDisposals(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pageintakemetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	metrics.IndexFailed(ctx, page, errors.New("no listener"))
	metrics.PageIndexed(ctx, page)

	body := exposition(t, registry)
	want := `corpustext_offered_pages_disposed_total{disposal="indexed"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output missing %q", want)
	}
	unwanted := `corpustext_offered_pages_disposed_total{disposal="index-failed"}`
	if strings.Contains(body, unwanted) {
		t.Errorf("metrics output contains %q", unwanted)
	}
}

func exposition(t *testing.T, registry *prometheus.Registry) string {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, request)

	return response.Body.String()
}
