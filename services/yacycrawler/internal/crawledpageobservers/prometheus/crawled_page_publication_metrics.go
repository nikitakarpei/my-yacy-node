// Package prometheus counts every crawled page the crawler writes twice over: how the write
// itself fared, so an operator can tell a page that reached the crawled-page stream from one
// that never left this process, and what the page stated about indexing, which holds whether
// the write succeeded or not.
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

type CrawledPagePublicationMetrics struct {
	publications *prometheusclient.CounterVec
	pages        *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *CrawledPagePublicationMetrics {
	metrics := &CrawledPagePublicationMetrics{
		publications: prometheusclient.NewCounterVec(
			prometheusclient.CounterOpts{
				Name: "yacycrawler_crawled_page_publications_total",
				Help: "Crawled page publications, by outcome.",
			}, []string{labelOutcome},
		),
		pages: prometheusclient.NewCounterVec(
			prometheusclient.CounterOpts{
				Name: "yacycrawler_crawled_pages_total",
				Help: "Crawled pages, by what the page states about indexing.",
			}, []string{labelIndexing},
		),
	}
	for _, outcome := range crawledPagePublicationOutcomes {
		metrics.publications.WithLabelValues(outcome)
	}
	for _, indexing := range pageIndexingStatements {
		metrics.pages.WithLabelValues(string(indexing))
	}
	registry.MustRegister(metrics.publications, metrics.pages)
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
	metrics.publications.WithLabelValues(outcome).Inc()
	metrics.pages.WithLabelValues(string(indexing)).Inc()
}
