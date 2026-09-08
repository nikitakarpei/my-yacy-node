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
	pageintakemetrics "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintakeobservers/prometheus"
)

const pageURL = "https://example.org/a"

func TestPageIntakeMetricsCountDisposalsAndRetries(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pageintakemetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.PageOffered(ctx, page)
	metrics.MarkdownStored(ctx, page)
	metrics.NoDocumentExtracted(ctx, page, cause)
	metrics.NoMarkdownDerived(ctx, page)
	metrics.MarkdownNotStored(ctx, page, cause)

	body := exposition(t, registry)
	for _, want := range []string{
		`corpusmarkdown_offered_pages_disposed_total{disposal="stored"} 1`,
		`corpusmarkdown_offered_pages_disposed_total{disposal="no-document-extracted"} 1`,
		`corpusmarkdown_offered_pages_disposed_total{disposal="no-markdown-derived"} 1`,
		`corpusmarkdown_pages_left_for_retry_total{cause="store-failed"} 1`,
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

	metrics.MarkdownNotStored(ctx, page, errors.New("no listener"))
	metrics.MarkdownStored(ctx, page)

	body := exposition(t, registry)
	want := `corpusmarkdown_offered_pages_disposed_total{disposal="stored"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output missing %q", want)
	}
	unwanted := `corpusmarkdown_offered_pages_disposed_total{disposal="store-failed"}`
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
