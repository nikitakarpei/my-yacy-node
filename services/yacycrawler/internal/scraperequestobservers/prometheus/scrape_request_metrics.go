// Package prometheus counts how every scrape request the crawler writes fared, so an
// operator can tell a request that reached the scrape service from one that never left
// this process.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomePublished        = "published"
	outcomeEncodingFailed   = "encoding_failed"
	outcomePublishingFailed = "publishing_failed"
)

var scrapeRequestPublicationOutcomes = []string{
	outcomePublished,
	outcomeEncodingFailed,
	outcomePublishingFailed,
}

type ScrapeRequestMetrics struct{ requestsProcessed *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *ScrapeRequestMetrics {
	metrics := &ScrapeRequestMetrics{requestsProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_scrape_requests_processed_total",
			Help: "Scrape requests processed, by outcome.",
		}, []string{labelOutcome},
	)}
	for _, outcome := range scrapeRequestPublicationOutcomes {
		metrics.requestsProcessed.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.requestsProcessed)
	return metrics
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestPublished(
	context.Context,
	canonicalurl.CanonicalURL,
) {
	metrics.count(outcomePublished)
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestEncodingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.count(outcomeEncodingFailed)
}

func (metrics *ScrapeRequestMetrics) ScrapeRequestPublishingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.count(outcomePublishingFailed)
}

func (metrics *ScrapeRequestMetrics) count(outcome string) {
	metrics.requestsProcessed.WithLabelValues(outcome).Inc()
}
