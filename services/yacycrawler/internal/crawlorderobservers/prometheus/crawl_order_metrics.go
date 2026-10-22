package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	labelOutcome    = "outcome"
	outcomeAccepted = "accepted"
	outcomeRejected = "rejected"
	outcomeReturned = "returned"
)

var crawlOrderOutcomes = []string{outcomeAccepted, outcomeRejected, outcomeReturned}

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
	for _, outcome := range crawlOrderOutcomes {
		metrics.ordersProcessed.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.ordersProcessed)
	return metrics
}

func (metrics *CrawlOrderMetrics) CrawlOrderReturned(context.Context, string, error) {
	metrics.ordersProcessed.WithLabelValues(outcomeReturned).Inc()
}

func (metrics *CrawlOrderMetrics) CrawlOrderRejected(context.Context, string, string, error) {
	metrics.ordersProcessed.WithLabelValues(outcomeRejected).Inc()
}

func (metrics *CrawlOrderMetrics) CrawlOrderAccepted(
	context.Context,
	string,
	crawlid.CrawlID,
	int,
) {
	metrics.ordersProcessed.WithLabelValues(outcomeAccepted).Inc()
}
