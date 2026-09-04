// Package prometheus counts how every crawled page the crawler writes fared, so an operator
// can tell a page that reached the fact stream from one that never left this process, and can
// see how many pages refused indexing.
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

	outcomeReported        = "reported"
	outcomeEncodingFailed  = "encoding_failed"
	outcomeReportingFailed = "reporting_failed"
)

var crawledPageReportOutcomes = []string{
	outcomeReported,
	outcomeEncodingFailed,
	outcomeReportingFailed,
}

var pageIndexingStatements = []crawledpagesjetstream.PageIndexing{
	crawledpagesjetstream.PageAllowsIndexing,
	crawledpagesjetstream.PageRefusesIndexing,
}

type CrawledPageMetrics struct{ pagesProcessed *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *CrawledPageMetrics {
	metrics := &CrawledPageMetrics{pagesProcessed: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_crawled_pages_processed_total",
			Help: "Crawled pages processed, by outcome and by what the page states about indexing.",
		}, []string{labelOutcome, labelIndexing},
	)}
	for _, outcome := range crawledPageReportOutcomes {
		for _, indexing := range pageIndexingStatements {
			metrics.pagesProcessed.WithLabelValues(outcome, string(indexing))
		}
	}
	registry.MustRegister(metrics.pagesProcessed)
	return metrics
}

func (metrics *CrawledPageMetrics) CrawledPageReported(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
) {
	metrics.count(outcomeReported, indexing)
}

func (metrics *CrawledPageMetrics) CrawledPageEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	_ error,
) {
	metrics.count(outcomeEncodingFailed, indexing)
}

func (metrics *CrawledPageMetrics) CrawledPageReportingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	_ error,
) {
	metrics.count(outcomeReportingFailed, indexing)
}

func (metrics *CrawledPageMetrics) count(
	outcome string,
	indexing crawledpagesjetstream.PageIndexing,
) {
	metrics.pagesProcessed.WithLabelValues(outcome, string(indexing)).Inc()
}
