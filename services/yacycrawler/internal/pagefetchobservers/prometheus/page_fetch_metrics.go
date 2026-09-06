package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomeSucceeded             = "succeeded"
	outcomeNotModified           = "not_modified"
	outcomeAccessRefused         = "access_refused"
	outcomeDeferred              = "deferred"
	outcomeRejected              = "rejected"
	outcomeRedirected            = "redirected"
	outcomeRedirectTargetInvalid = "redirect_target_invalid"
	outcomeOversized             = "oversized"
	outcomeFailed                = "failed"
	outcomeCanceled              = "canceled"
)

var pageFetchOutcomes = []string{
	outcomeSucceeded,
	outcomeNotModified,
	outcomeAccessRefused,
	outcomeDeferred,
	outcomeRejected,
	outcomeRedirected,
	outcomeRedirectTargetInvalid,
	outcomeOversized,
	outcomeFailed,
	outcomeCanceled,
}

type PageFetchMetrics struct {
	pagesProcessed   *prometheusclient.CounterVec
	pageFetchSeconds prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *PageFetchMetrics {
	metrics := &PageFetchMetrics{
		pagesProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_fetches_processed_total",
			Help: "Page fetches processed, by outcome.",
		}, []string{labelOutcome}),
		pageFetchSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name: "yacycrawler_page_fetch_duration_seconds",
			Help: "Page fetch duration in seconds.",
		}),
	}
	for _, outcome := range pageFetchOutcomes {
		metrics.pagesProcessed.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.pagesProcessed, metrics.pageFetchSeconds)
	return metrics
}

func (m *PageFetchMetrics) PageFetchSucceeded(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeSucceeded, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchNotModified(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeNotModified, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchAccessRefused(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeAccessRefused, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchDeferred(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ time.Duration,
) {
	m.record(outcomeDeferred, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRejected(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeRejected, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRedirected(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	m.record(outcomeRedirected, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRedirectTargetInvalid(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ error,
) {
	m.record(outcomeRedirectTargetInvalid, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRefusedOversizedPage(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeOversized, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchFailed(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ error,
) {
	m.record(outcomeFailed, fetchDuration)
}

func (m *PageFetchMetrics) PageFetchCanceled(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record(outcomeCanceled, fetchDuration)
}

func (m *PageFetchMetrics) record(outcome string, fetchDuration time.Duration) {
	m.pagesProcessed.WithLabelValues(outcome).Inc()
	m.pageFetchSeconds.Observe(fetchDuration.Seconds())
}
