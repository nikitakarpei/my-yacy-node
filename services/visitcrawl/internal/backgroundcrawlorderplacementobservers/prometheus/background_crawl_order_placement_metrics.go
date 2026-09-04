package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome    = "outcome"
	outcomeAccepted = "accepted"
	outcomeRefused  = "refused"
)

type BackgroundCrawlOrderPlacementMetrics struct {
	placements *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *BackgroundCrawlOrderPlacementMetrics {
	metrics := &BackgroundCrawlOrderPlacementMetrics{
		placements: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "visitcrawl_background_crawl_order_placements_total",
			Help: "Crawl orders offered for background placement, by outcome.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.placements)
	return metrics
}

func (metrics *BackgroundCrawlOrderPlacementMetrics) CrawlOrderPlacementAccepted(
	_ context.Context,
	_ string,
) {
	metrics.placements.WithLabelValues(outcomeAccepted).Inc()
}

func (metrics *BackgroundCrawlOrderPlacementMetrics) CrawlOrderPlacementRefused(
	_ context.Context,
	_ string,
) {
	metrics.placements.WithLabelValues(outcomeRefused).Inc()
}
