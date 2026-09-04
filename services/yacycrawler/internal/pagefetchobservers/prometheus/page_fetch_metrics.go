package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const labelOutcome = "outcome"

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
	registry.MustRegister(metrics.pagesProcessed, metrics.pageFetchSeconds)
	return metrics
}

func (m *PageFetchMetrics) PageFetchSucceeded(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record("succeeded", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchNotModified(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record("not_modified", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchAccessRefused(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record("access_refused", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchDeferred(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ time.Duration,
) {
	m.record("deferred", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRejected(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record("rejected", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchLandedURLInvalid(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ error,
) {
	m.record("landed_url_invalid", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchRefusedOversizedPage(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration,
) {
	m.record("oversized", fetchDuration)
}

func (m *PageFetchMetrics) PageFetchFailed(
	_ context.Context, _ canonicalurl.CanonicalURL, fetchDuration time.Duration, _ error,
) {
	m.record("failed", fetchDuration)
}

func (m *PageFetchMetrics) record(outcome string, fetchDuration time.Duration) {
	m.pagesProcessed.WithLabelValues(outcome).Inc()
	m.pageFetchSeconds.Observe(fetchDuration.Seconds())
}
