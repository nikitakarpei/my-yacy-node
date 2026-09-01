package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	scrapeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/scrapeprogressobservers/prometheus"
)

func TestScrapeProgressMetricsCountAttemptsAndAdmissions(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := scrapeprogressobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.test/page")
	cause := errors.New("scrape failed")

	metrics.ScrapeRequestInvalid(ctx)
	metrics.OriginFetchFailed(ctx, "message", pageURL, cause)
	metrics.OriginFetchDeferred(ctx, "message", pageURL, time.Second)
	metrics.NothingToScrape(ctx, "message", pageURL)
	metrics.DocumentExtractionFailed(ctx, "message", pageURL, cause)
	metrics.NoIndexDerived(ctx, "message", pageURL)
	metrics.URLMetadataAdmissionBusy(ctx, "message", pageURL)
	metrics.URLMetadataAdmissionFailed(ctx, "message", pageURL, cause)
	metrics.PostingsAdmissionBusy(ctx, "message", pageURL, 11)
	metrics.PostingsAdmissionFailed(ctx, "message", pageURL, 13, cause)
	metrics.URLMetadataAdmitted(ctx, "message", pageURL)
	metrics.PostingsAdmitted(ctx, "message", pageURL, 17)
	metrics.ScrapeRequestCompleted(ctx, "message", pageURL)

	body := exposition(t, registry)
	for _, attemptOutcome := range []string{
		"completed",
		"fetch_failed",
		"fetch_deferred",
		"nothing_to_scrape",
		"document_extraction_failed",
		"no_index_derived",
		"url_metadata_admission_busy",
		"url_metadata_admission_failed",
		"postings_admission_busy",
		"postings_admission_failed",
		"invalid_message",
	} {
		want := `scraperequestintake_attempts_total{outcome="` + attemptOutcome + `"} 1`
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
	for _, want := range []string{
		"scraperequestintake_url_metadata_admitted_total 1",
		"scraperequestintake_postings_admitted_total 17",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestScrapeProgressMetricsDoNotExposeMessageOrURLLabels(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := scrapeprogressobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://secret.example/page")

	metrics.URLMetadataAdmitted(context.Background(), "secret-message", pageURL)
	metrics.PostingsAdmitted(context.Background(), "secret-message", pageURL, 1)
	metrics.ScrapeRequestCompleted(context.Background(), "secret-message", pageURL)

	body := exposition(t, registry)
	for _, secret := range []string{"secret.example", "secret-message"} {
		if strings.Contains(body, secret) {
			t.Errorf("metrics output contains %q", secret)
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
