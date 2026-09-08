// Package prometheus counts what became of every page the scrape service offered to this
// corpus, so an operator can tell a page that carries no document from one that carries no
// markdown, and either of those from a corpus that turns the write down and leaves the page
// for a later retry.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOfferedPageDisposal = "disposal"
	labelRetryCause          = "cause"
)

const (
	disposalStored              = "stored"
	disposalNoDocumentExtracted = "no-document-extracted"
	disposalNoMarkdownDerived   = "no-markdown-derived"
)

var offeredPageDisposals = []string{
	disposalStored,
	disposalNoDocumentExtracted,
	disposalNoMarkdownDerived,
}

const retryCauseStoreFailed = "store-failed"

var retryCauses = []string{
	retryCauseStoreFailed,
}

type PageIntakeMetrics struct {
	offeredPagesDisposed *prometheus.CounterVec
	pagesLeftForRetry    *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *PageIntakeMetrics {
	metrics := &PageIntakeMetrics{
		offeredPagesDisposed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusmarkdown_offered_pages_disposed_total",
			Help: "Offered pages, by the disposal that ended their intake.",
		}, []string{labelOfferedPageDisposal}),
		pagesLeftForRetry: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_left_for_retry_total",
			Help: "Offered pages the corpus could not store yet, by cause.",
		}, []string{labelRetryCause}),
	}
	for _, disposal := range offeredPageDisposals {
		metrics.offeredPagesDisposed.WithLabelValues(disposal)
	}
	for _, cause := range retryCauses {
		metrics.pagesLeftForRetry.WithLabelValues(cause)
	}
	registry.MustRegister(
		metrics.offeredPagesDisposed,
		metrics.pagesLeftForRetry,
	)
	return metrics
}

func (m *PageIntakeMetrics) PageOffered(_ context.Context, _ canonicalurl.CanonicalURL) {
}

func (m *PageIntakeMetrics) NoDocumentExtracted(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.dispose(disposalNoDocumentExtracted)
}

func (m *PageIntakeMetrics) NoMarkdownDerived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoMarkdownDerived)
}

func (m *PageIntakeMetrics) MarkdownNotStored(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.leaveForRetry(retryCauseStoreFailed)
}

func (m *PageIntakeMetrics) MarkdownStored(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.dispose(disposalStored)
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}

func (m *PageIntakeMetrics) leaveForRetry(cause string) {
	m.pagesLeftForRetry.WithLabelValues(cause).Inc()
}
