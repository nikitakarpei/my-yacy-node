package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome     = "outcome"
	outcomePlaced    = "placed"
	outcomeFailed    = "failed"
	outcomeSaturated = "saturated"
)

type CrawlOrderPlacementMetrics struct {
	placementsProcessed *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *CrawlOrderPlacementMetrics {
	metrics := &CrawlOrderPlacementMetrics{
		placementsProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "visitcrawl_crawl_order_placements_processed_total",
			Help: "Crawl order placement attempts processed, by outcome.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.placementsProcessed)
	return metrics
}

func (metrics *CrawlOrderPlacementMetrics) CrawlOrderPlaced(
	_ context.Context,
	_ string,
) {
	metrics.placementsProcessed.WithLabelValues(outcomePlaced).Inc()
}

func (metrics *CrawlOrderPlacementMetrics) CrawlOrderPlacementFailed(
	_ context.Context,
	_ string,
	_ error,
) {
	metrics.placementsProcessed.WithLabelValues(outcomeFailed).Inc()
}

func (metrics *CrawlOrderPlacementMetrics) CrawlOrderPlacementSkippedBecauseSaturated(
	context.Context,
	string,
) {
	metrics.placementsProcessed.WithLabelValues(outcomeSaturated).Inc()
}
