package prometheus_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	placementmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/backgroundcrawlorderplacementobservers/prometheus"
)

const placementsMetric = "visitcrawl_background_crawl_order_placements_total"

func TestBackgroundCrawlOrderPlacementMetricsCountEveryOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := placementmetricsprometheus.New(registry)

	metrics.CrawlOrderPlacementAccepted(context.Background(), "order-1")
	metrics.CrawlOrderPlacementRefused(context.Background(), "order-2")

	expected := `
# HELP ` + placementsMetric + ` Crawl orders offered for background placement, by outcome.
# TYPE ` + placementsMetric + ` counter
` + placementsMetric + `{outcome="accepted"} 1
` + placementsMetric + `{outcome="refused"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		placementsMetric,
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
