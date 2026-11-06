package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageintakeobservers/prometheus"
)

func TestPageIntakeMetricsCountDisposalsRetriesAndBuiltPostings(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := pageintakeobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("intake failed")

	metrics.OfferedPageInvalid(ctx)
	metrics.PageOffered(ctx, "message", pageURL)
	metrics.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	metrics.MediaTypeUnsupported(ctx, "message", pageURL, "application/pdf")
	metrics.NoIndexDerived(ctx, "message", pageURL)
	metrics.PageRWINotPublished(ctx, "message", pageURL)
	metrics.PageRWIBuilt(ctx, "message", pageURL, 17)

	body := exposition(t, registry)
	for _, want := range []string{
		`pagerwi_pageintake_offered_pages_disposed_total{disposal="built"} 1`,
		`pagerwi_pageintake_offered_pages_disposed_total{disposal="document_extraction_failed"} 1`,
		`pagerwi_pageintake_offered_pages_disposed_total{disposal="media_type_unsupported"} 1`,
		`pagerwi_pageintake_offered_pages_disposed_total{disposal="no_index_derived"} 1`,
		`pagerwi_pageintake_offered_pages_disposed_total{disposal="invalid_message"} 1`,
		`pagerwi_pageintake_pages_left_for_retry_total{cause="page_rwi_not_published"} 1`,
		"pagerwi_pageintake_postings_built_total 17",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestPageIntakeMetricsDoNotExposeMessageOrURLLabels(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := pageintakeobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://secret.example/page")

	metrics.PageRWINotPublished(context.Background(), "secret-message", pageURL)
	metrics.PageRWIBuilt(context.Background(), "secret-message", pageURL, 1)

	body := exposition(t, registry)
	for _, secret := range []string{"secret.example", "secret-message"} {
		if strings.Contains(body, secret) {
			t.Errorf("metrics output contains %q", secret)
		}
	}
}

func TestPageIntakeMetricsKeepRetriedPagesOutOfDisposals(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := pageintakeobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")

	metrics.PageRWINotPublished(ctx, "message", pageURL)

	body := exposition(t, registry)
	unwanted := `pagerwi_pageintake_offered_pages_disposed_total{disposal="built"} 1`
	if strings.Contains(body, unwanted) {
		t.Errorf("metrics output contains %q", unwanted)
	}
}

func exposition(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, request)

	return response.Body.String()
}
