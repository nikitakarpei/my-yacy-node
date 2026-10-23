package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scraperequestobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/scraperequestobservers/prometheus"
)

const metricName = "crawlscrapebridge_scrape_requests_processed_total"

func TestScrapeRequestMetricsCountEveryOutcome(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := scraperequestobserversprometheus.New(registry)

	metrics.ScrapeRequestPublished(context.Background(), canonicalurl.CanonicalURL{})
	metrics.ScrapeRequestPublished(context.Background(), canonicalurl.CanonicalURL{})
	metrics.ScrapeRequestNotPublished(
		context.Background(), canonicalurl.CanonicalURL{}, errors.New("no stream"),
	)

	expected := `
# HELP crawlscrapebridge_scrape_requests_processed_total Scrape requests processed, by outcome.
# TYPE crawlscrapebridge_scrape_requests_processed_total counter
crawlscrapebridge_scrape_requests_processed_total{outcome="not_published"} 1
crawlscrapebridge_scrape_requests_processed_total{outcome="published"} 2
`
	if err := testutil.GatherAndCompare(
		registry, strings.NewReader(expected), metricName,
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
