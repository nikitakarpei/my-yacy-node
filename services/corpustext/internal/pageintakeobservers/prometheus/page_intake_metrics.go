// Package prometheus counts what became of every page the scrape service offered to this
// corpus, so an operator can tell a page that carries no document from one that carries no
// readable text, and either of those from a search index that turns the write down.
package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOfferedPageDisposal = "disposal"

	disposalIndexed               = "indexed"
	disposalNoDocumentExtracted   = "no-document-extracted"
	disposalNoReadableTextDerived = "no-readable-text-derived"
	disposalIndexFailed           = "index-failed"
)

var offeredPageDisposals = []string{
	disposalIndexed,
	disposalNoDocumentExtracted,
	disposalNoReadableTextDerived,
	disposalIndexFailed,
}

type PageIntakeMetrics struct {
	pagesOffered         prometheus.Counter
	offeredPagesDisposed *prometheus.CounterVec
	indexDurationSecs    prometheus.Histogram
}

func New(registry prometheus.Registerer) *PageIntakeMetrics {
	metrics := &PageIntakeMetrics{
		pagesOffered: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpustext_pages_offered_total",
			Help: "Pages the scrape service offered to this corpus.",
		}),
		offeredPagesDisposed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpustext_offered_pages_disposed_total",
			Help: "Offered pages, by how the corpus disposed of each one.",
		}, []string{labelOfferedPageDisposal}),
		indexDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "corpustext_index_duration_seconds",
			Help:    "Search-index write duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	for _, disposal := range offeredPageDisposals {
		metrics.offeredPagesDisposed.WithLabelValues(disposal)
	}
	registry.MustRegister(
		metrics.pagesOffered,
		metrics.offeredPagesDisposed,
		metrics.indexDurationSecs,
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

func (m *PageIntakeMetrics) NoReadableTextDerived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoReadableTextDerived)
}

func (m *PageIntakeMetrics) IndexObserved(_ context.Context, elapsed time.Duration) {
	m.indexDurationSecs.Observe(elapsed.Seconds())
}

func (m *PageIntakeMetrics) IndexFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.dispose(disposalIndexFailed)
}

func (m *PageIntakeMetrics) PageIndexed(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.dispose(disposalIndexed)
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}
