package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const labelOutcome = "outcome"

type CrawlOrderMetrics struct {
	ordersProcessed *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *CrawlOrderMetrics {
	metrics := &CrawlOrderMetrics{ordersProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_crawl_orders_processed_total",
			Help: "Crawl orders processed, by outcome.",
		}, []string{labelOutcome},
	)}
	registry.MustRegister(metrics.ordersProcessed)
	return metrics
}

func (metrics *CrawlOrderMetrics) CrawlOrderReturned(context.Context, string, error) {
	metrics.ordersProcessed.WithLabelValues("returned").Inc()
}

func (metrics *CrawlOrderMetrics) CrawlOrderAccepted(context.Context, string, int) {
	metrics.ordersProcessed.WithLabelValues("accepted").Inc()
}
