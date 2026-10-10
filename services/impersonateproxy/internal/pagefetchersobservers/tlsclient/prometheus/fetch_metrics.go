// Package prometheus counts the page fetches as Prometheus metrics.
package prometheus

import (
	"context"
	"strconv"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type FetchMetrics struct {
	fetchedPages     *prometheusclient.CounterVec
	failedFetches    prometheusclient.Counter
	cancelledFetches prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *FetchMetrics {
	metrics := &FetchMetrics{
		fetchedPages: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "impersonateproxy_fetched_pages_total",
			Help: "Pages that the origin answered, by response status.",
		}, []string{"status"}),
		failedFetches: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "impersonateproxy_failed_fetches_total",
			Help: "Page fetches that ended without a response.",
		}),
		cancelledFetches: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "impersonateproxy_cancelled_fetches_total",
			Help: "Page fetches that the client closed before the response.",
		}),
	}
	registry.MustRegister(metrics.fetchedPages, metrics.failedFetches, metrics.cancelledFetches)
	return metrics
}

func (m *FetchMetrics) PageFetched(_ context.Context, _ string, status int) {
	m.fetchedPages.WithLabelValues(strconv.Itoa(status)).Inc()
}

func (m *FetchMetrics) FetchFailed(context.Context, string, error) {
	m.failedFetches.Inc()
}

func (m *FetchMetrics) FetchCancelled(context.Context, string) {
	m.cancelledFetches.Inc()
}
