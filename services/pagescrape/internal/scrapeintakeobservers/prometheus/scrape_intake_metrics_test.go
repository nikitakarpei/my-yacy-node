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
	scrapeintakemetrics "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeintakeobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const pageURL = "https://example.org/a"

func TestScrapeIntakeMetricsCountDisposalsAndRetries(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := scrapeintakemetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.ScrapeRequestInvalid(ctx, "message", cause)
	metrics.ScrapeRequestReceived(ctx, page)
	metrics.OriginFetchFailed(ctx, page, cause)
	metrics.PageOffered(ctx, page, page)
	metrics.ScrapeDeferred(ctx, page, time.Second)
	metrics.ScrapeFailed(ctx, page, pagescrapecontract.Oversized)
	metrics.PageNotOffered(ctx, page, cause)
	metrics.ScrapeScheduleFailed(ctx, page, cause)

	body := exposition(t, registry)
	for _, want := range []string{
		`pagescrape_scrape_requests_disposed_total{disposal="unreadable"} 1`,
		`pagescrape_scrape_requests_disposed_total{disposal="offered"} 1`,
		`pagescrape_scrape_requests_disposed_total{disposal="scheduled"} 1`,
		`pagescrape_scrape_requests_disposed_total{disposal="oversized"} 1`,
		`pagescrape_scrape_requests_left_for_retry_total{cause="offer-refused"} 1`,
		`pagescrape_scrape_requests_left_for_retry_total{cause="schedule-refused"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func TestScrapeIntakeMetricsKeepRetriedRequestsOutOfDisposals(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := scrapeintakemetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	metrics.PageNotOffered(ctx, page, errors.New("no listener"))
	metrics.PageOffered(ctx, page, page)

	body := exposition(t, registry)
	want := `pagescrape_scrape_requests_disposed_total{disposal="offered"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output missing %q", want)
	}
	unwanted := `pagescrape_scrape_requests_disposed_total{disposal="offer-refused"}`
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
