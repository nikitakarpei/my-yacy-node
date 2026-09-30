// Package prometheus counts the requests refused for their response headers
// deadline as Prometheus metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/deadlineenforcing"
)

type DeadlineMetrics struct {
	refusals *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *DeadlineMetrics {
	metrics := &DeadlineMetrics{
		refusals: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "spamproxy_deadline_refusals_total",
			Help: "Requests refused because their response headers could not arrive in time, by reason.",
		}, []string{"reason"}),
	}
	registry.MustRegister(metrics.refusals)
	return metrics
}

func (m *DeadlineMetrics) RequestRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason deadlineenforcing.RefusalReason,
) {
	m.refusals.WithLabelValues(string(reason)).Inc()
}
