// Package prometheus counts how every crawled page the crawler writes fared, so an operator
// can tell a page that reached the crawled-page stream from one that never left this process,
// and can see how many pages refused indexing.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
)

const (
	labelOutcome  = "outcome"
	labelIndexing = "indexing"

	outcomePublished        = "published"
	outcomeEncodingFailed   = "encoding_failed"
	outcomePublishingFailed = "publishing_failed"
)

var crawledPagePublicationOutcomes = []string{
	outcomePublished,
	outcomeEncodingFailed,
	outcomePublishingFailed,
}

var pageIndexingStatements = []crawledpagesjetstream.PageIndexing{
	crawledpagesjetstream.PageAllowsIndexing,
	crawledpagesjetstream.PageRefusesIndexing,
}

type CrawledPagePublicationMetrics struct{ pagesProcessed *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *CrawledPagePublicationMetrics {
	metrics := &CrawledPagePublicationMetrics{pagesProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_crawled_pages_processed_total",
			Help: "Crawled pages processed, by outcome and by what the page states about indexing.",
		}, []string{labelOutcome, labelIndexing},
	)}
	for _, outcome := range crawledPagePublicationOutcomes {
		for _, indexing := range pageIndexingStatements {
			metrics.pagesProcessed.WithLabelValues(outcome, string(indexing))
		}
	}
	registry.MustRegister(metrics.pagesProcessed)
	return metrics
}

func (metrics *CrawledPagePublicationMetrics) CrawledPagePublished(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
) {
	metrics.count(outcomePublished, indexing)
}

func (metrics *CrawledPagePublicationMetrics) CrawledPageEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	_ error,
) {
	metrics.count(outcomeEncodingFailed, indexing)
}

func (metrics *CrawledPagePublicationMetrics) CrawledPagePublishingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	_ error,
) {
	metrics.count(outcomePublishingFailed, indexing)
}

func (metrics *CrawledPagePublicationMetrics) count(
	outcome string,
	indexing crawledpagesjetstream.PageIndexing,
) {
	metrics.pagesProcessed.WithLabelValues(outcome, string(indexing)).Inc()
}
