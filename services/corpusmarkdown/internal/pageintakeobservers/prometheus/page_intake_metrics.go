// Package prometheus counts what became of every page the scrape service offered to this
// corpus, so an operator can tell a page that carries no document from one that carries no
// markdown, and either of those from a corpus that turns the write down.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOfferedPageDisposal = "disposal"

	disposalStored              = "stored"
	disposalNoDocumentExtracted = "no-document-extracted"
	disposalNoMarkdownDerived   = "no-markdown-derived"
	disposalStoreFailed         = "store-failed"
)

var offeredPageDisposals = []string{
	disposalStored,
	disposalNoDocumentExtracted,
	disposalNoMarkdownDerived,
	disposalStoreFailed,
}

type PageIntakeMetrics struct {
	pagesOffered         prometheus.Counter
	offeredPagesDisposed *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *PageIntakeMetrics {
	metrics := &PageIntakeMetrics{
		pagesOffered: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusmarkdown_pages_offered_total",
			Help: "Pages the scrape service offered to this corpus.",
		}),
		offeredPagesDisposed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusmarkdown_offered_pages_disposed_total",
			Help: "Offered pages, by how the corpus disposed of each one.",
		}, []string{labelOfferedPageDisposal}),
	}
	for _, disposal := range offeredPageDisposals {
		metrics.offeredPagesDisposed.WithLabelValues(disposal)
	}
	registry.MustRegister(
		metrics.pagesOffered,
		metrics.offeredPagesDisposed,
	)
	return metrics
}

func (m *PageIntakeMetrics) PageOffered(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.pagesOffered.Inc()
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
	m.dispose(disposalStoreFailed)
}

func (m *PageIntakeMetrics) MarkdownStored(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.dispose(disposalStored)
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}
