package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	crawlordermetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlorderobservers/prometheus"
)

func TestCrawlOrdersCountOneOutcomeEach(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crawlordermetricsprometheus.New(registry)
	ctx := context.Background()

	metrics.CrawlOrderAccepted(ctx, "o1", crawlid.CrawlID{1}, 2)
	metrics.CrawlOrderRejected(ctx, "message", "o2", errors.New("unreadable"))
	metrics.CrawlOrderReturned(ctx, "o3", errors.New("unavailable"))

	expected := `
# HELP yacycrawler_crawl_orders_processed_total Crawl orders processed, by outcome.
# TYPE yacycrawler_crawl_orders_processed_total counter
yacycrawler_crawl_orders_processed_total{outcome="accepted"} 1
yacycrawler_crawl_orders_processed_total{outcome="rejected"} 1
yacycrawler_crawl_orders_processed_total{outcome="returned"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}
