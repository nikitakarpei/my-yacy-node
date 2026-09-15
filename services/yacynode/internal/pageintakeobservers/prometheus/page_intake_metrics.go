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
	disposalIndexed                  = "indexed"
	disposalDocumentExtractionFailed = "document_extraction_failed"
	disposalNoIndexDerived           = "no_index_derived"
	disposalInvalidMessage           = "invalid_message"
)

var offeredPageDisposals = []string{
	disposalIndexed,
	disposalDocumentExtractionFailed,
	disposalNoIndexDerived,
	disposalInvalidMessage,
}

const (
	retryCausePageAdmissionBusy   = "page_admission_busy"
	retryCausePageAdmissionFailed = "page_admission_failed"
)

var retryCauses = []string{
	retryCausePageAdmissionBusy,
	retryCausePageAdmissionFailed,
}

type PageIntakeMetrics struct {
	offeredPagesDisposed *prometheusclient.CounterVec
	pagesLeftForRetry    *prometheusclient.CounterVec
	pagesAdmitted        prometheusclient.Counter
	postingsAdmitted     prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageIntakeMetrics {
	offeredPagesDisposed := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacynode_pageintake_offered_pages_disposed_total",
			Help: "Offered pages, by the disposal that ended their intake.",
		},
		[]string{labelOfferedPageDisposal},
	)
	for _, disposal := range offeredPageDisposals {
		offeredPagesDisposed.WithLabelValues(disposal)
	}
	pagesLeftForRetry := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacynode_pageintake_pages_left_for_retry_total",
			Help: "Offered pages the node could not take in yet, by cause.",
		},
		[]string{labelRetryCause},
	)
	for _, cause := range retryCauses {
		pagesLeftForRetry.WithLabelValues(cause)
	}
	pagesAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_pages_admitted_total",
		Help: "Offered pages the node admitted as one replacement of what it held.",
	})
	postingsAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_postings_admitted_total",
		Help: "Posting admissions accepted while taking in offered pages.",
	})
	registry.MustRegister(
		offeredPagesDisposed,
		pagesLeftForRetry,
		pagesAdmitted,
		postingsAdmitted,
	)

	return &PageIntakeMetrics{
		offeredPagesDisposed: offeredPagesDisposed,
		pagesLeftForRetry:    pagesLeftForRetry,
		pagesAdmitted:        pagesAdmitted,
		postingsAdmitted:     postingsAdmitted,
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

func (m *PageIntakeMetrics) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoIndexDerived)
}

func (m *PageIntakeMetrics) PageAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	m.pagesAdmitted.Inc()
	m.postingsAdmitted.Add(float64(amountOfPostings))
}

func (m *PageIntakeMetrics) PageAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
) {
	m.leaveForRetry(retryCausePageAdmissionBusy)
}

func (m *PageIntakeMetrics) PageAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
	error,
) {
	m.leaveForRetry(retryCausePageAdmissionFailed)
}

func (m *PageIntakeMetrics) PageIndexed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalIndexed)
}

func (m *PageIntakeMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}

func (m *PageIntakeMetrics) leaveForRetry(cause string) {
	m.pagesLeftForRetry.WithLabelValues(cause).Inc()
}
