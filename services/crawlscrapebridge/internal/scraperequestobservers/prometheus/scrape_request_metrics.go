// Package prometheus counts how every scrape request the bridge publishes fared, so an
// operator can tell a crawled page that reached the scrape service from one that waits.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomePublished    = "published"
	outcomeNotPublished = "not_published"
)

type ScrapeRequestMetrics struct{ requestsProcessed *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *ScrapeRequestMetrics {
	metrics := &ScrapeRequestMetrics{requestsProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "crawlscrapebridge_scrape_requests_processed_total",
			Help: "Scrape requests processed, by outcome.",
		}, []string{labelOutcome},
	)}
	for _, outcome := range []string{outcomePublished, outcomeNotPublished} {
		metrics.requestsProcessed.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.requestsProcessed)
	return metrics
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestPublished(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	metrics.requestsProcessed.WithLabelValues(outcomePublished).Inc()
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestNotPublished(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	metrics.requestsProcessed.WithLabelValues(outcomeNotPublished).Inc()
}
