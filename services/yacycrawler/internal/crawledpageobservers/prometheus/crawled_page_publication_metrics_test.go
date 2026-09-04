package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawledpagepublicationmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpageobservers/prometheus"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
)

func TestCrawledPagePublicationMetricsCountEveryOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crawledpagepublicationmetricsprometheus.New(registry)
	pageURL := canonicalurl.CanonicalURL{}

	metrics.CrawledPagePublished(
		context.Background(), pageURL, crawledpagesjetstream.PageAllowsIndexing,
	)
	metrics.CrawledPagePublished(
		context.Background(), pageURL, crawledpagesjetstream.PageRefusesIndexing,
	)
	metrics.CrawledPageEncodingFailed(
		context.Background(), pageURL, crawledpagesjetstream.PageAllowsIndexing,
		errors.New("unwritable"),
	)
	metrics.CrawledPagePublishingFailed(
		context.Background(), pageURL, crawledpagesjetstream.PageAllowsIndexing,
		errors.New("no responders"),
	)

	expected := `
# HELP yacycrawler_crawled_pages_processed_total Crawled pages processed, by outcome and by what the page states about indexing.
# TYPE yacycrawler_crawled_pages_processed_total counter
yacycrawler_crawled_pages_processed_total{indexing="allowed",outcome="encoding_failed"} 1
yacycrawler_crawled_pages_processed_total{indexing="allowed",outcome="published"} 1
yacycrawler_crawled_pages_processed_total{indexing="allowed",outcome="publishing_failed"} 1
yacycrawler_crawled_pages_processed_total{indexing="refused",outcome="encoding_failed"} 0
yacycrawler_crawled_pages_processed_total{indexing="refused",outcome="published"} 1
yacycrawler_crawled_pages_processed_total{indexing="refused",outcome="publishing_failed"} 0
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacycrawler_crawled_pages_processed_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestEveryPublicationOutcomeReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheus.NewRegistry()
	crawledpagepublicationmetricsprometheus.New(registry)

	if got := testutil.CollectAndCount(
		registry,
		"yacycrawler_crawled_pages_processed_total",
	); got != 6 {
		t.Fatalf("publication outcome series = %d, want 6", got)
	}
}
