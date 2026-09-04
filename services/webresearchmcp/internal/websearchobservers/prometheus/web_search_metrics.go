// Package prometheus counts the searches the service serves and the searches it cannot,
// so an operator can tell a busy service from one whose search engine is away.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type WebSearchMetrics struct {
	searchesServed prometheus.Counter
	searchFailures prometheus.Counter
}

func New(registry prometheus.Registerer) *WebSearchMetrics {
	metrics := &WebSearchMetrics{
		searchesServed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_searches_served_total",
			Help: "Searches answered with the results of the configured search engine.",
		}),
		searchFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_search_failures_total",
			Help: "Searches the configured search engine did not answer.",
		}),
	}
	registry.MustRegister(metrics.searchesServed, metrics.searchFailures)
	return metrics
}

func (m *WebSearchMetrics) SearchServed(_ context.Context, _ string, _ int) {
	m.searchesServed.Inc()
}

func (m *WebSearchMetrics) SearchFailed(_ context.Context, _ string, _ error) {
	m.searchFailures.Inc()
}
