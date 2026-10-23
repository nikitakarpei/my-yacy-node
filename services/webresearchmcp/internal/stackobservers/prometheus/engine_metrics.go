// Package prometheus counts the searches each search engine of the stack fails, so an
// operator can tell which engine is away.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type EngineMetrics struct {
	engineFailures *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *EngineMetrics {
	metrics := &EngineMetrics{
		engineFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webresearchmcp_search_engine_failures_total",
			Help: "Searches a search engine of the stack did not answer, by engine.",
		}, []string{"engine"}),
	}
	registry.MustRegister(metrics.engineFailures)
	return metrics
}

func (m *EngineMetrics) EngineFailed(_ context.Context, engineName string, _ string, _ error) {
	m.engineFailures.WithLabelValues(engineName).Inc()
}
