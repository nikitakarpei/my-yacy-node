package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	publicationmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderpublicationobservers/prometheus"
)

const publicationsMetric = "visitcrawl_crawl_order_publications_total"

func TestCrawlOrderPublicationMetricsCountEveryOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := publicationmetricsprometheus.New(registry)

	metrics.CrawlOrderPublished(context.Background(), "order-1", "yacy.crawl.orders")
	metrics.CrawlOrderPublishingFailed(
		context.Background(),
		"order-2",
		"yacy.crawl.orders",
		errors.New("broker down"),
	)
	metrics.CrawlOrderEncodingFailed(context.Background(), "order-3", errors.New("bad order"))

	expected := `
# HELP ` + publicationsMetric + ` Crawl order publications to the orders subject, by outcome.
# TYPE ` + publicationsMetric + ` counter
` + publicationsMetric + `{outcome="encoding_failed"} 1
` + publicationsMetric + `{outcome="published"} 1
` + publicationsMetric + `{outcome="publishing_failed"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		publicationsMetric,
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
