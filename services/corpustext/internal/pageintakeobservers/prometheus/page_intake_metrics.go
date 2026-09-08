// Package prometheus counts what became of every page the scrape service offered to this
// corpus, so an operator can tell a page that carries no document from one that carries no
// readable text, and either of those from a search index that turns the write down and
// leaves the page for a later retry.
package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOfferedPageDisposal = "disposal"
	labelRetryCause          = "cause"
)

const (
	disposalIndexed               = "indexed"
	disposalNoDocumentExtracted   = "no-document-extracted"
	disposalNoReadableTextDerived = "no-readable-text-derived"
)

var offeredPageDisposals = []string{
	disposalIndexed,
	disposalNoDocumentExtracted,
	disposalNoReadableTextDerived,
}

const retryCauseIndexFailed = "index-failed"

var retryCauses = []string{
	retryCauseIndexFailed,
}

type PageIntakeMetrics struct {
	offeredPagesDisposed *prometheus.CounterVec
	pagesLeftForRetry    *prometheus.CounterVec
	indexDurationSecs    prometheus.Histogram
}

func New(registry prometheus.Registerer) *PageIntakeMetrics {
	metrics := &PageIntakeMetrics{
		offeredPagesDisposed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpustext_offered_pages_disposed_total",
			Help: "Offered pages, by the disposal that ended their intake.",
		}, []string{labelOfferedPageDisposal}),
		pagesLeftForRetry: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpustext_pages_left_for_retry_total",
			Help: "Offered pages the corpus could not index yet, by cause.",
		}, []string{labelRetryCause}),
		indexDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "corpustext_index_duration_seconds",
			Help:    "Search-index write duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
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
		metrics.indexDurationSecs,
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

func (m *PageIntakeMetrics) NoReadableTextDerived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoReadableTextDerived)
}

func (m *PageIntakeMetrics) IndexWriteEnded(_ context.Context, elapsed time.Duration) {
	m.indexDurationSecs.Observe(elapsed.Seconds())
}

func (m *PageIntakeMetrics) IndexFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.leaveForRetry(retryCauseIndexFailed)
}

func (m *PageIntakeMetrics) PageIndexed(_ context.Context, _ canonicalurl.CanonicalURL) {
	m.dispose(disposalIndexed)
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}

func (m *PageIntakeMetrics) leaveForRetry(cause string) {
	m.pagesLeftForRetry.WithLabelValues(cause).Inc()
}
