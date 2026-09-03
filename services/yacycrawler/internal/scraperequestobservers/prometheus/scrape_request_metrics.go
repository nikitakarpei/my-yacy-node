package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const labelOutcome = "outcome"

type ScrapeRequestMetrics struct{ requestsProcessed *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *ScrapeRequestMetrics {
	metrics := &ScrapeRequestMetrics{requestsProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_scrape_requests_processed_total",
			Help: "Scrape requests processed, by outcome.",
		}, []string{labelOutcome},
	)}
	registry.MustRegister(metrics.requestsProcessed)
	return metrics
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestPublished(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	metrics.requestsProcessed.WithLabelValues("published").Inc()
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestPublicationFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.requestsProcessed.WithLabelValues("failed").Inc()
}
