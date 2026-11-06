// Package prometheus exports each page intake fact as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOfferedPageDisposal = "disposal"
	labelRetryCause          = "cause"
)

const (
	disposalBuilt                    = "built"
	disposalDocumentExtractionFailed = "document_extraction_failed"
	disposalMediaTypeUnsupported     = "media_type_unsupported"
	disposalNoIndexDerived           = "no_index_derived"
	disposalInvalidMessage           = "invalid_message"
)

var offeredPageDisposals = []string{
	disposalBuilt,
	disposalDocumentExtractionFailed,
	disposalMediaTypeUnsupported,
	disposalNoIndexDerived,
	disposalInvalidMessage,
}

const retryCausePageRWINotPublished = "page_rwi_not_published"

type PageIntakeMetrics struct {
	offeredPagesDisposed *prometheusclient.CounterVec
	pagesLeftForRetry    *prometheusclient.CounterVec
	postingsBuilt        prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageIntakeMetrics {
	offeredPagesDisposed := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "pagerwi_pageintake_offered_pages_disposed_total",
			Help: "Offered pages, by the disposal that ended their intake.",
		},
		[]string{labelOfferedPageDisposal},
	)
	for _, disposal := range offeredPageDisposals {
		offeredPagesDisposed.WithLabelValues(disposal)
	}
	pagesLeftForRetry := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "pagerwi_pageintake_pages_left_for_retry_total",
			Help: "Offered pages the service could not finish yet, by cause.",
		},
		[]string{labelRetryCause},
	)
	pagesLeftForRetry.WithLabelValues(retryCausePageRWINotPublished)
	postingsBuilt := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "pagerwi_pageintake_postings_built_total",
		Help: "Postings of the RWIs the service built and published.",
	})
	registry.MustRegister(offeredPagesDisposed, pagesLeftForRetry, postingsBuilt)

	return &PageIntakeMetrics{
		offeredPagesDisposed: offeredPagesDisposed,
		pagesLeftForRetry:    pagesLeftForRetry,
		postingsBuilt:        postingsBuilt,
	}
}

func (m *PageIntakeMetrics) OfferedPageInvalid(context.Context) {
	m.dispose(disposalInvalidMessage)
}

func (m *PageIntakeMetrics) PageOffered(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
}

func (m *PageIntakeMetrics) DocumentExtractionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.dispose(disposalDocumentExtractionFailed)
}

func (m *PageIntakeMetrics) MediaTypeUnsupported(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	string,
) {
	m.dispose(disposalMediaTypeUnsupported)
}

func (m *PageIntakeMetrics) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoIndexDerived)
}

func (m *PageIntakeMetrics) PageRWINotPublished(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.pagesLeftForRetry.WithLabelValues(retryCausePageRWINotPublished).Inc()
}

func (m *PageIntakeMetrics) PageRWIBuilt(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	m.dispose(disposalBuilt)
	m.postingsBuilt.Add(float64(amountOfPostings))
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}
