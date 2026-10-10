// Package prometheus counts the facts of the proxy intake as Prometheus
// metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintake"
)

type IntakeMetrics struct {
	refusedRequests     *prometheusclient.CounterVec
	incompleteResponses *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *IntakeMetrics {
	metrics := &IntakeMetrics{
		refusedRequests: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "impersonateproxy_refused_requests_total",
			Help: "Requests that the proxy refused, by the part it refused.",
		}, []string{"part"}),
		incompleteResponses: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "impersonateproxy_incomplete_responses_total",
			Help: "Responses that ended before the whole body, by cause.",
		}, []string{"cause"}),
	}
	registry.MustRegister(metrics.refusedRequests, metrics.incompleteResponses)
	return metrics
}

func (m *IntakeMetrics) MethodRefused(context.Context, string) {
	m.refusedRequests.WithLabelValues("method").Inc()
}

func (m *IntakeMetrics) TargetRefused(context.Context, string) {
	m.refusedRequests.WithLabelValues("target").Inc()
}

func (m *IntakeMetrics) ResponseLeftIncomplete(
	_ context.Context,
	_ string,
	incompleteResponseCause proxyintake.IncompleteResponseCause,
	_ error,
) {
	m.incompleteResponses.WithLabelValues(string(incompleteResponseCause)).Inc()
}
