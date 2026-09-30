// Package prometheus counts the facts of the relay as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type RelayMetrics struct {
	incompleteResponses *prometheusclient.CounterVec
	closedRequests      prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RelayMetrics {
	metrics := &RelayMetrics{
		incompleteResponses: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_incomplete_responses_total",
			Help: "Responses that ended before the whole body, by cause.",
		}, []string{"cause"}),
		closedRequests: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "spamproxy_client_closed_requests_total",
			Help: "Requests that the client closed before the response headers.",
		}),
	}
	registry.MustRegister(metrics.incompleteResponses, metrics.closedRequests)
	return metrics
}

func (m *RelayMetrics) ResponseLeftIncomplete(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	incompleteResponseCause requestrelay.IncompleteResponseCause,
	_ error,
) {
	m.incompleteResponses.WithLabelValues(string(incompleteResponseCause)).Inc()
}

func (m *RelayMetrics) ClientClosedRequest(context.Context, canonicalurl.CanonicalURL) {
	m.closedRequests.Inc()
}
