// Package prometheus counts the failed upstream requests as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/failurereporting"
)

type FailureMetrics struct {
	failures *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *FailureMetrics {
	metrics := &FailureMetrics{
		failures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_upstream_request_failures_total",
			Help: "Upstream requests that failed, by the step that failed.",
		}, []string{"step"}),
	}
	registry.MustRegister(metrics.failures)
	return metrics
}

func (m *FailureMetrics) RoundTripFailed(
	_ context.Context,
	_ string,
	step failurereporting.Step,
	_ error,
) {
	m.failures.WithLabelValues(string(step)).Inc()
}
