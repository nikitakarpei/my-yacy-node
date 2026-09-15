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
	pageintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintakeobservers/prometheus"
)

func TestPageIntakeMetricsCountDisposalsRetriesAndAdmissions(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := pageintakeobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("intake failed")

	metrics.OfferedPageInvalid(ctx)
	metrics.PageOffered(ctx, "message", pageURL)
	metrics.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	metrics.NoIndexDerived(ctx, "message", pageURL)
	metrics.PageAdmissionBusy(ctx, "message", pageURL, 11)
	metrics.PageAdmissionFailed(ctx, "message", pageURL, 13, cause)
	metrics.PageAdmitted(ctx, "message", pageURL, 17)
	metrics.PageIndexed(ctx, "message", pageURL)

	body := exposition(t, registry)
	for _, disposal := range []string{
		"indexed",
		"document_extraction_failed",
		"no_index_derived",
		"invalid_message",
	} {
		want := `yacynode_pageintake_offered_pages_disposed_total{disposal="` + disposal + `"} 1`
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
	for _, cause := range []string{
		"page_admission_busy",
		"page_admission_failed",
	} {
		want := `yacynode_pageintake_pages_left_for_retry_total{cause="` + cause + `"} 1`
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
	for _, want := range []string{
		"yacynode_pageintake_pages_admitted_total 1",
		"yacynode_pageintake_postings_admitted_total 17",
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

	metrics.PageAdmitted(context.Background(), "secret-message", pageURL, 1)
	metrics.PageIndexed(context.Background(), "secret-message", pageURL)

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

	metrics.PageAdmissionBusy(ctx, "message", pageURL, 3)
	metrics.PageIndexed(ctx, "message", pageURL)

	body := exposition(t, registry)
	want := `yacynode_pageintake_offered_pages_disposed_total{disposal="indexed"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output missing %q", want)
	}
	for _, cause := range []string{"page_admission_busy"} {
		unwanted := `yacynode_pageintake_offered_pages_disposed_total{disposal="` + cause + `"}`
		if strings.Contains(body, unwanted) {
			t.Errorf("metrics output contains %q", unwanted)
		}
	}
}

func exposition(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, request)

	return response.Body.String()
}
