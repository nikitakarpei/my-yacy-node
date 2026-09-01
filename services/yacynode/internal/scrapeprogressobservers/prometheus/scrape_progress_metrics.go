// Package prometheus exports scrape request progress as Prometheus metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const labelAttemptOutcome = "outcome"

const (
	attemptCompleted                  = "completed"
	attemptFetchFailed                = "fetch_failed"
	attemptFetchDeferred              = "fetch_deferred"
	attemptNothingToScrape            = "nothing_to_scrape"
	attemptDocumentExtractionFailed   = "document_extraction_failed"
	attemptNoIndexDerived             = "no_index_derived"
	attemptURLMetadataAdmissionBusy   = "url_metadata_admission_busy"
	attemptURLMetadataAdmissionFailed = "url_metadata_admission_failed"
	attemptPostingsAdmissionBusy      = "postings_admission_busy"
	attemptPostingsAdmissionFailed    = "postings_admission_failed"
	attemptInvalidMessage             = "invalid_message"
)

var attemptOutcomes = []string{
	attemptCompleted,
	attemptFetchFailed,
	attemptFetchDeferred,
	attemptNothingToScrape,
	attemptDocumentExtractionFailed,
	attemptNoIndexDerived,
	attemptURLMetadataAdmissionBusy,
	attemptURLMetadataAdmissionFailed,
	attemptPostingsAdmissionBusy,
	attemptPostingsAdmissionFailed,
	attemptInvalidMessage,
}

type ScrapeProgressMetrics struct {
	attempts            *prometheusclient.CounterVec
	urlMetadataAdmitted prometheusclient.Counter
	postingsAdmitted    prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *ScrapeProgressMetrics {
	attempts := prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "scraperequestintake_attempts_total",
			Help: "Scrape request delivery attempts, by terminal outcome.",
		},
		[]string{labelAttemptOutcome},
	)
	for _, attemptOutcome := range attemptOutcomes {
		attempts.WithLabelValues(attemptOutcome)
	}
	urlMetadataAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "scraperequestintake_url_metadata_admitted_total",
		Help: "URL metadata admissions accepted while processing scrape requests.",
	})
	postingsAdmitted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "scraperequestintake_postings_admitted_total",
		Help: "Posting admissions accepted while processing scrape requests.",
	})
	registry.MustRegister(attempts, urlMetadataAdmitted, postingsAdmitted)

	return &ScrapeProgressMetrics{
		attempts:            attempts,
		urlMetadataAdmitted: urlMetadataAdmitted,
		postingsAdmitted:    postingsAdmitted,
	}
}

func (m *ScrapeProgressMetrics) ScrapeRequestInvalid(context.Context) {
	m.observeAttempt(attemptInvalidMessage)
}

func (m *ScrapeProgressMetrics) OriginFetchFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.observeAttempt(attemptFetchFailed)
}

func (m *ScrapeProgressMetrics) OriginFetchDeferred(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	time.Duration,
) {
	m.observeAttempt(attemptFetchDeferred)
}

func (m *ScrapeProgressMetrics) NothingToScrape(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.observeAttempt(attemptNothingToScrape)
}

func (m *ScrapeProgressMetrics) DocumentExtractionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.observeAttempt(attemptDocumentExtractionFailed)
}

func (m *ScrapeProgressMetrics) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.observeAttempt(attemptNoIndexDerived)
}

func (m *ScrapeProgressMetrics) URLMetadataAdmitted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.urlMetadataAdmitted.Inc()
}

func (m *ScrapeProgressMetrics) URLMetadataAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	m.observeAttempt(attemptURLMetadataAdmissionBusy)
}

func (m *ScrapeProgressMetrics) URLMetadataAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	m.observeAttempt(attemptURLMetadataAdmissionFailed)
}

func (m *ScrapeProgressMetrics) PostingsAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	m.postingsAdmitted.Add(float64(postings))
}

func (m *ScrapeProgressMetrics) PostingsAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
) {
	m.observeAttempt(attemptPostingsAdmissionBusy)
}

func (m *ScrapeProgressMetrics) PostingsAdmissionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	int,
	error,
) {
	m.observeAttempt(attemptPostingsAdmissionFailed)
}

func (m *ScrapeProgressMetrics) ScrapeRequestCompleted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
) {
	m.observeAttempt(attemptCompleted)
}

func (m *ScrapeProgressMetrics) observeAttempt(attemptOutcome string) {
	m.attempts.WithLabelValues(attemptOutcome).Inc()
}
