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
	retryCauseURLMetadataAdmissionBusy   = "url_metadata_admission_busy"
	retryCauseURLMetadataAdmissionFailed = "url_metadata_admission_failed"
	retryCausePostingsAdmissionBusy      = "postings_admission_busy"
	retryCausePostingsAdmissionFailed    = "postings_admission_failed"
)

var retryCauses = []string{
	retryCauseURLMetadataAdmissionBusy,
	retryCauseURLMetadataAdmissionFailed,
	retryCausePostingsAdmissionBusy,
	retryCausePostingsAdmissionFailed,
}

type PageIntakeMetrics struct {
	offeredPagesDisposed *prometheusclient.CounterVec
	pagesLeftForRetry    *prometheusclient.CounterVec
	urlMetadataAdmitted  prometheusclient.Counter
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
	urlMetadataAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_url_metadata_admitted_total",
		Help: "URL metadata admissions accepted while taking in offered pages.",
	})
	postingsAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_postings_admitted_total",
		Help: "Posting admissions accepted while taking in offered pages.",
	})
	registry.MustRegister(
		offeredPagesDisposed,
		pagesLeftForRetry,
		urlMetadataAdmitted,
		postingsAdmitted,
	)

	return &PageIntakeMetrics{
		offeredPagesDisposed: offeredPagesDisposed,
		pagesLeftForRetry:    pagesLeftForRetry,
		urlMetadataAdmitted:  urlMetadataAdmitted,
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

func (m *PageIntakeMetrics) URLMetadataAdmitted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.urlMetadataAdmitted.Inc()
}

func (m *PageIntakeMetrics) URLMetadataAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.leaveForRetry(retryCauseURLMetadataAdmissionBusy)
}

func (m *PageIntakeMetrics) URLMetadataAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.leaveForRetry(retryCauseURLMetadataAdmissionFailed)
}

func (m *PageIntakeMetrics) PostingsAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	m.postingsAdmitted.Add(float64(postings))
}

func (m *PageIntakeMetrics) PostingsAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
) {
	m.leaveForRetry(retryCausePostingsAdmissionBusy)
}

func (m *PageIntakeMetrics) PostingsAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
	error,
) {
	m.leaveForRetry(retryCausePostingsAdmissionFailed)
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
