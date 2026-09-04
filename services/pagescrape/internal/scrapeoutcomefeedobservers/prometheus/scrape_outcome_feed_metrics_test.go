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
	scrapeoutcomefeedmetrics "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeoutcomefeedobservers/prometheus"
)

const (
	pageURL        = "https://example.org/a"
	outcomeSubject = "yacy.scrape.outcomes.abc.failure"
	receiptSubject = "yacy.scrape.receipts.abc.kept"
)

func TestScrapeOutcomeFeedMetricsCountsEveryOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := scrapeoutcomefeedmetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.ScrapeFailureAnnounced(ctx, page, outcomeSubject)
	metrics.ScrapeFailureEncodingFailed(ctx, page, cause)
	metrics.ScrapeFailurePublishingFailed(ctx, page, outcomeSubject, cause)
	metrics.ScrapeFailureConfirmationFailed(ctx, page, outcomeSubject, cause)
	metrics.IntakeReceiptCarried(ctx, receiptSubject, outcomeSubject)
	metrics.IntakeReceiptSubjectUnreadable(ctx, receiptSubject, cause)
	metrics.IntakeReceiptNotCarried(ctx, receiptSubject, outcomeSubject, cause)

	req := httptest.NewRequestWithContext(ctx, "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		`pagescrape_scrape_failure_announcements_total{outcome="announced"} 1`,
		`pagescrape_scrape_failure_announcements_total{outcome="encoding_failed"} 1`,
		`pagescrape_scrape_failure_announcements_total{outcome="publishing_failed"} 1`,
		`pagescrape_scrape_failure_announcements_total{outcome="confirmation_failed"} 1`,
		`pagescrape_intake_receipt_carriages_total{outcome="carried"} 1`,
		`pagescrape_intake_receipt_carriages_total{outcome="subject_unreadable"} 1`,
		`pagescrape_intake_receipt_carriages_total{outcome="not_carried"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
