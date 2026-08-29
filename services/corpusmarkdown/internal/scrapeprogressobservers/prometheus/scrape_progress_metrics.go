// Package prometheus counts what became of every scrape request the markdown corpus took
// on, so an operator can tell an origin that serves nothing from a derivation that yields
// nothing, and a corpus write that fails from a redirection write that fails.
package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeProgressMetrics struct {
	scrapeRequestsReceived            prometheus.Counter
	originFetchFailures               prometheus.Counter
	originFetchDeferrals              prometheus.Counter
	documentExtractionFailures        prometheus.Counter
	markdownCorpusWriteFailures       prometheus.Counter
	redirectionRecordWriteFailures    prometheus.Counter
	nothingToScrape                   prometheus.Counter
	noMarkdownDerived                 prometheus.Counter
	pagesStored                       prometheus.Counter
	scrapeOutcomeAnnouncementFailures prometheus.Counter
}

func New(registry prometheus.Registerer) *ScrapeProgressMetrics {
	metrics := &ScrapeProgressMetrics{
		scrapeRequestsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_scrape_requests_received_total",
			Help: "Scrape requests received.",
		}),
		originFetchFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_origin_fetch_failures_total",
			Help: "Fetches from the origin that failed and returned the scrape request for redelivery.",
		}),
		originFetchDeferrals: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_origin_fetch_deferrals_total",
			Help: "Fetches the origin asked to be held back, returning the scrape request for later.",
		}),
		documentExtractionFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_document_extraction_failures_total",
			Help: "Scrape requests given up on because no document could be extracted from the page.",
		}),
		markdownCorpusWriteFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_markdown_corpus_write_failures_total",
			Help: "Markdown corpus writes that failed and returned the scrape request for redelivery.",
		}),
		redirectionRecordWriteFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_redirection_record_write_failures_total",
			Help: "Redirection record writes that failed and returned the scrape request for redelivery.",
		}),
		nothingToScrape: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_nothing_to_scrape_total",
			Help: "Scrape requests given up on because the origin served no content to scrape.",
		}),
		noMarkdownDerived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_no_markdown_derived_total",
			Help: "Scrape requests given up on because the fetched page yielded no markdown.",
		}),
		pagesStored: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_stored_total",
			Help: "Pages written to the object store.",
		}),
		scrapeOutcomeAnnouncementFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_scrape_outcome_announcement_failures_total",
			Help: "Settled scrape requests whose outcome could not be announced to waiting callers.",
		}),
	}
	registry.MustRegister(
		metrics.scrapeRequestsReceived,
		metrics.originFetchFailures,
		metrics.originFetchDeferrals,
		metrics.documentExtractionFailures,
		metrics.markdownCorpusWriteFailures,
		metrics.redirectionRecordWriteFailures,
		metrics.nothingToScrape,
		metrics.noMarkdownDerived,
		metrics.pagesStored,
		metrics.scrapeOutcomeAnnouncementFailures,
	)
	return metrics
}

func (m *ScrapeProgressMetrics) ScrapeRequestReceived(_ context.Context) {
	m.scrapeRequestsReceived.Inc()
}

func (m *ScrapeProgressMetrics) OriginFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.originFetchFailures.Inc()
}

func (m *ScrapeProgressMetrics) OriginFetchDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	m.originFetchDeferrals.Inc()
}

func (m *ScrapeProgressMetrics) NothingToScrape(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.nothingToScrape.Inc()
}

func (m *ScrapeProgressMetrics) DocumentExtractionFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.documentExtractionFailures.Inc()
}

func (m *ScrapeProgressMetrics) NoMarkdownDerived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
	m.noMarkdownDerived.Inc()
}

func (m *ScrapeProgressMetrics) MarkdownCorpusWriteFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.markdownCorpusWriteFailures.Inc()
}

func (m *ScrapeProgressMetrics) RedirectionRecordWriteFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.redirectionRecordWriteFailures.Inc()
}

func (m *ScrapeProgressMetrics) MarkdownStored(_ context.Context, _, _ canonicalurl.CanonicalURL) {
	m.pagesStored.Inc()
}

func (m *ScrapeProgressMetrics) ScrapeOutcomeAnnouncementFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.scrapeOutcomeAnnouncementFailures.Inc()
}
