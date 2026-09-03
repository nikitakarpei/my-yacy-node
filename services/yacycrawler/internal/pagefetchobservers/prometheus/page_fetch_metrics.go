package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
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

func (metrics *PageFetchMetrics) PageFetchCompleted(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	status pagefetch.FetchStatus,
	duration time.Duration,
) {
	metrics.record(pageFetchOutcome(status), duration)
}

func (metrics *PageFetchMetrics) PageFetchFailed(
	_ context.Context, _ canonicalurl.CanonicalURL, duration time.Duration, _ error,
) {
	metrics.record("error", duration)
}

func (metrics *PageFetchMetrics) record(outcome string, duration time.Duration) {
	metrics.pagesProcessed.WithLabelValues(outcome).Inc()
	metrics.pageFetchSeconds.Observe(duration.Seconds())
}

func pageFetchOutcome(status pagefetch.FetchStatus) string {
	switch status {
	case pagefetch.FetchSucceeded:
		return "succeeded"
	case pagefetch.FetchNotModified:
		return "not_modified"
	case pagefetch.FetchAccessRefused:
		return "access_refused"
	case pagefetch.FetchDeferred:
		return "deferred"
	case pagefetch.FetchRejected:
		return "rejected"
	case pagefetch.FetchLandedURLInvalid:
		return "landed_url_invalid"
	case pagefetch.FetchOversized:
		return "oversized"
	case pagefetch.FetchFailed:
		return "failed"
	default:
		return "unknown"
	}
}
