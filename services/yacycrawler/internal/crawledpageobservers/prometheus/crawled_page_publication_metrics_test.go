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

func metricsAfterEveryOutcome(t *testing.T) *prometheus.Registry {
	t.Helper()
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
		context.Background(), pageURL, crawledpagesjetstream.PageRefusesIndexing,
		errors.New("unwritable"),
	)
	metrics.CrawledPagePublishingFailed(
		context.Background(), pageURL, crawledpagesjetstream.PageAllowsIndexing,
		errors.New("no responders"),
	)
	return registry
}

func TestPublicationsCountEveryOutcome(t *testing.T) {
	expected := `
# HELP yacycrawler_crawled_page_publications_total Crawled page publications, by outcome.
# TYPE yacycrawler_crawled_page_publications_total counter
yacycrawler_crawled_page_publications_total{outcome="encoding_failed"} 1
yacycrawler_crawled_page_publications_total{outcome="published"} 2
yacycrawler_crawled_page_publications_total{outcome="publishing_failed"} 1
`
	if err := testutil.GatherAndCompare(
		metricsAfterEveryOutcome(t),
		strings.NewReader(expected),
		"yacycrawler_crawled_page_publications_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestAPageCountsUnderItsIndexingStatementWhateverBecameOfTheWrite(t *testing.T) {
	expected := `
# HELP yacycrawler_crawled_pages_total Crawled pages, by what the page states about indexing.
# TYPE yacycrawler_crawled_pages_total counter
yacycrawler_crawled_pages_total{indexing="allowed"} 2
yacycrawler_crawled_pages_total{indexing="refused"} 2
`
	if err := testutil.GatherAndCompare(
		metricsAfterEveryOutcome(t),
		strings.NewReader(expected),
		"yacycrawler_crawled_pages_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestEveryOutcomeAndStatementReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheus.NewRegistry()
	crawledpagepublicationmetricsprometheus.New(registry)

	if got := testutil.CollectAndCount(
		registry,
		"yacycrawler_crawled_page_publications_total",
		"yacycrawler_crawled_pages_total",
	); got != 5 {
		t.Fatalf("crawled page series = %d, want 5", got)
	}
}
