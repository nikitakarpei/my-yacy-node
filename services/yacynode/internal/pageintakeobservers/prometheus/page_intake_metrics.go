// Package prometheus exports each page intake fact as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelPageRWIDisposal = "disposal"
	labelRetryCause      = "cause"
)

const (
	disposalIndexed              = "indexed"
	disposalInvalidMessage       = "invalid_message"
	disposalPostingsRootMismatch = "postings_root_mismatch"
)

var pageRWIDisposals = []string{
	disposalIndexed,
	disposalInvalidMessage,
	disposalPostingsRootMismatch,
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
	pageRWIsDisposed  *prometheusclient.CounterVec
	pagesLeftForRetry *prometheusclient.CounterVec
	pagesAdmitted     prometheusclient.Counter
	postingsAdmitted  prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageIntakeMetrics {
	pageRWIsDisposed := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacynode_pageintake_page_rwis_disposed_total",
			Help: "RWIs of pages, by the disposal that ended their intake.",
		},
		[]string{labelPageRWIDisposal},
	)
	for _, disposal := range pageRWIDisposals {
		pageRWIsDisposed.WithLabelValues(disposal)
	}
	pagesLeftForRetry := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "yacynode_pageintake_pages_left_for_retry_total",
			Help: "RWIs of pages the node could not take in yet, by cause.",
		},
		[]string{labelRetryCause},
	)
	for _, cause := range retryCauses {
		pagesLeftForRetry.WithLabelValues(cause)
	}
	pagesAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_pages_admitted_total",
		Help: "RWIs of pages the node admitted as one replacement of what it held.",
	})
	postingsAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacynode_pageintake_postings_admitted_total",
		Help: "Posting admissions accepted while taking in RWIs of pages.",
	})
	registry.MustRegister(
		pageRWIsDisposed,
		pagesLeftForRetry,
		pagesAdmitted,
		postingsAdmitted,
	)

	return &PageIntakeMetrics{
		pageRWIsDisposed:  pageRWIsDisposed,
		pagesLeftForRetry: pagesLeftForRetry,
		pagesAdmitted:     pagesAdmitted,
		postingsAdmitted:  postingsAdmitted,
	}
}

func (m *PageIntakeMetrics) PageRWIInvalid(context.Context) {
	m.dispose(disposalInvalidMessage)
}

func (m *PageIntakeMetrics) PageRWIReceived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
}

func (m *PageIntakeMetrics) PostingsRootMismatched(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.dispose(disposalPostingsRootMismatch)
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
	m.pageRWIsDisposed.WithLabelValues(disposal).Inc()
}

func (m *PageIntakeMetrics) leaveForRetry(cause string) {
	m.pagesLeftForRetry.WithLabelValues(cause).Inc()
}
