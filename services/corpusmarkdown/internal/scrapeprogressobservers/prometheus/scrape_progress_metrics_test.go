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

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scrapeprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/scrapeprogressobservers/prometheus"
)

func TestScrapeProgressMetricsCountsEveryFactItIsTold(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := scrapeprogressobserversprometheus.New(registry)
	requestedURL := urlOf(t, "https://example.test/page")
	markdownURL := urlOf(t, "https://example.test/moved")
	cause := errors.New("told alongside the fact")

	metrics.ScrapeRequestReceived(ctx)
	metrics.OriginFetchFailed(ctx, requestedURL, cause)
	metrics.OriginFetchDeferred(ctx, requestedURL, time.Second)
	metrics.NothingToScrape(ctx, requestedURL)
	metrics.DocumentExtractionFailed(ctx, requestedURL, markdownURL, cause)
	metrics.NoMarkdownDerived(ctx, requestedURL, markdownURL)
	metrics.MarkdownCorpusWriteFailed(ctx, markdownURL, cause)
	metrics.RedirectionRecordWriteFailed(ctx, requestedURL, markdownURL, cause)
	metrics.MarkdownStored(ctx, requestedURL, markdownURL)

	body := exposition(t, registry)
	for _, want := range []string{
		"corpusmarkdown_scrape_requests_received_total 1",
		"corpusmarkdown_origin_fetch_failures_total 1",
		"corpusmarkdown_origin_fetch_deferrals_total 1",
		"corpusmarkdown_document_extraction_failures_total 1",
		"corpusmarkdown_markdown_corpus_write_failures_total 1",
		"corpusmarkdown_redirection_record_write_failures_total 1",
		"corpusmarkdown_nothing_to_scrape_total 1",
		"corpusmarkdown_no_markdown_derived_total 1",
		"corpusmarkdown_pages_stored_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}

func TestScrapeProgressMetricsExposesNoUrlLabel(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := scrapeprogressobserversprometheus.New(registry)

	secret := urlOf(t, "https://example.test/secret")

	metrics.MarkdownStored(ctx, secret, secret)
	metrics.OriginFetchFailed(ctx, secret, errors.New("failed"))
	metrics.DocumentExtractionFailed(ctx, secret, secret, errors.New("failed"))

	if body := exposition(t, registry); strings.Contains(body, "example.test") {
		t.Fatalf("metrics output labels a counter by url, got:\n%s", body)
	}
}

func exposition(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)
	return rec.Body.String()
}

func urlOf(t *testing.T, raw string) canonicalurl.CanonicalURL {
	t.Helper()
	parsed, err := canonicalurl.CanonicalURLOf(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}
