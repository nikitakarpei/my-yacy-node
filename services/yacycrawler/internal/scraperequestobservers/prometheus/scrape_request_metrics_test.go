package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scraperequestmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/scraperequestobservers/prometheus"
)

func TestScrapeRequestMetricsCountEveryPublicationOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := scraperequestmetricsprometheus.New(registry)
	pageURL := canonicalurl.CanonicalURL{}

	metrics.ScrapeRequestPublished(context.Background(), pageURL)
	metrics.ScrapeRequestMarshalingFailed(
		context.Background(), pageURL, errors.New("unwritable"),
	)
	metrics.ScrapeRequestPublishingFailed(
		context.Background(), pageURL, errors.New("no responders"),
	)

	expected := `
# HELP yacycrawler_scrape_requests_processed_total Scrape requests processed, by outcome.
# TYPE yacycrawler_scrape_requests_processed_total counter
yacycrawler_scrape_requests_processed_total{outcome="marshaling_failed"} 1
yacycrawler_scrape_requests_processed_total{outcome="published"} 1
yacycrawler_scrape_requests_processed_total{outcome="publishing_failed"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacycrawler_scrape_requests_processed_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestEveryPublicationOutcomeReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheus.NewRegistry()
	scraperequestmetricsprometheus.New(registry)

	if got := testutil.CollectAndCount(
		registry,
		"yacycrawler_scrape_requests_processed_total",
	); got != 3 {
		t.Fatalf("publication outcome series = %d, want 3", got)
	}
}
