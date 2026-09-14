package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome            = "outcome"
	outcomePublished        = "published"
	outcomePublishingFailed = "publishing_failed"
	outcomeEncodingFailed   = "encoding_failed"
)

type CrawlOrderPublicationMetrics struct {
	publications *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *CrawlOrderPublicationMetrics {
	metrics := &CrawlOrderPublicationMetrics{
		publications: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "visitcrawl_crawl_order_publications_total",
			Help: "Crawl order publications to the orders subject, by outcome.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.publications)
	return metrics
}

func (metrics *CrawlOrderPublicationMetrics) CrawlOrderPublished(
	_ context.Context,
	_ string,
	_ string,
) {
	metrics.publications.WithLabelValues(outcomePublished).Inc()
}

func (metrics *CrawlOrderPublicationMetrics) CrawlOrderPublishingFailed(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) {
	metrics.publications.WithLabelValues(outcomePublishingFailed).Inc()
}

func (metrics *CrawlOrderPublicationMetrics) CrawlOrderEncodingFailed(
	_ context.Context,
	_ string,
	_ error,
) {
	metrics.publications.WithLabelValues(outcomeEncodingFailed).Inc()
}
