// Package prometheus exports page intake progress as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const labelOfferedPageDisposal = "disposal"

const (
	disposalIndexed                    = "indexed"
	disposalDocumentExtractionFailed   = "document_extraction_failed"
	disposalNoIndexDerived             = "no_index_derived"
	disposalURLMetadataAdmissionBusy   = "url_metadata_admission_busy"
	disposalURLMetadataAdmissionFailed = "url_metadata_admission_failed"
	disposalPostingsAdmissionBusy      = "postings_admission_busy"
	disposalPostingsAdmissionFailed    = "postings_admission_failed"
	disposalInvalidMessage             = "invalid_message"
)

var offeredPageDisposals = []string{
	disposalIndexed,
	disposalDocumentExtractionFailed,
	disposalNoIndexDerived,
	disposalURLMetadataAdmissionBusy,
	disposalURLMetadataAdmissionFailed,
	disposalPostingsAdmissionBusy,
	disposalPostingsAdmissionFailed,
	disposalInvalidMessage,
}

type IntakeProgressMetrics struct {
	pagesOffered          prometheusclient.Counter
	offeredPagesDisposed  *prometheusclient.CounterVec
	urlMetadataAdmitted   prometheusclient.Counter
	postingsAdmitted      prometheusclient.Counter
	intakeReceiptFailures prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *IntakeProgressMetrics {
	pagesOffered := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "pageintake_pages_offered_total",
		Help: "Pages the scrape service offered to this node.",
	})
	offeredPagesDisposed := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "pageintake_offered_pages_disposed_total",
			Help: "Offered page deliveries, by terminal disposal.",
		},
		[]string{labelOfferedPageDisposal},
	)
	for _, disposal := range offeredPageDisposals {
		offeredPagesDisposed.WithLabelValues(disposal)
	}
	urlMetadataAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "pageintake_url_metadata_admitted_total",
		Help: "URL metadata admissions accepted while taking in offered pages.",
	})
	postingsAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "pageintake_postings_admitted_total",
		Help: "Posting admissions accepted while taking in offered pages.",
	})
	intakeReceiptFailures := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "pageintake_intake_receipt_failures_total",
		Help: "Intake receipts that reached no caller waiting for the page.",
	})
	registry.MustRegister(
		pagesOffered,
		offeredPagesDisposed,
		urlMetadataAdmitted,
		postingsAdmitted,
		intakeReceiptFailures,
	)

	return &IntakeProgressMetrics{
		pagesOffered:          pagesOffered,
		offeredPagesDisposed:  offeredPagesDisposed,
		urlMetadataAdmitted:   urlMetadataAdmitted,
		postingsAdmitted:      postingsAdmitted,
		intakeReceiptFailures: intakeReceiptFailures,
	}
}

func (m *IntakeProgressMetrics) OfferedPageInvalid(context.Context) {
	m.dispose(disposalInvalidMessage)
}

func (m *IntakeProgressMetrics) PageOffered(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.pagesOffered.Inc()
}

func (m *IntakeProgressMetrics) DocumentExtractionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.dispose(disposalDocumentExtractionFailed)
}

func (m *IntakeProgressMetrics) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.dispose(disposalNoIndexDerived)
}

func (m *IntakeProgressMetrics) URLMetadataAdmitted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.urlMetadataAdmitted.Inc()
}

func (m *IntakeProgressMetrics) URLMetadataAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.dispose(disposalURLMetadataAdmissionBusy)
}

func (m *IntakeProgressMetrics) URLMetadataAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.dispose(disposalURLMetadataAdmissionFailed)
}

func (m *IntakeProgressMetrics) PostingsAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	m.postingsAdmitted.Add(float64(postings))
}

func (m *IntakeProgressMetrics) PostingsAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
) {
	m.dispose(disposalPostingsAdmissionBusy)
}

func (m *IntakeProgressMetrics) PostingsAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
	error,
) {
	m.dispose(disposalPostingsAdmissionFailed)
}

func (m *IntakeProgressMetrics) IntakeReceiptNotSent(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.intakeReceiptFailures.Inc()
}

func (m *IntakeProgressMetrics) PageIndexed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalIndexed)
}

func (m *IntakeProgressMetrics) dispose(disposal string) {
	m.offeredPagesDisposed.WithLabelValues(disposal).Inc()
}
