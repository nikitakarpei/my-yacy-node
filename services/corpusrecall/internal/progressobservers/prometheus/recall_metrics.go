// Package prometheus observes the progress of recalls and exposes it to Prometheus.
package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
)

type RecallMetrics struct {
	registry         *prometheus.Registry
	requestsAccepted prometheus.Counter
	requestsRejected prometheus.Counter
	recalled         *prometheus.CounterVec
	unavailable      *prometheus.CounterVec
}

func NewRecallMetrics() *RecallMetrics {
	registry := prometheus.NewRegistry()
	metrics := &RecallMetrics{
		registry: registry,
		requestsAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusrecall_requests_accepted_total",
			Help: "Recall requests admitted for retrieval.",
		}),
		requestsRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "corpusrecall_requests_rejected_total",
			Help: "Recall requests rejected because the in-flight limit was reached.",
		}),
		recalled: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusrecall_representations_recalled_total",
			Help: "Representations returned from the corpus, by kind.",
		}, []string{"kind"}),
		unavailable: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusrecall_representations_unavailable_total",
			Help: "Representations reported unavailable at the recall limit, by kind.",
		}, []string{"kind"}),
	}
	registry.MustRegister(
		metrics.requestsAccepted,
		metrics.requestsRejected,
		metrics.recalled,
		metrics.unavailable,
	)
	return metrics
}

func (m *RecallMetrics) RequestAccepted() { m.requestsAccepted.Inc() }
func (m *RecallMetrics) RequestRejected() { m.requestsRejected.Inc() }

func (m *RecallMetrics) RepresentationRecalled(kind recall.RepresentationKind) {
	m.recalled.WithLabelValues(string(kind)).Inc()
}

func (m *RecallMetrics) RepresentationUnavailable(kind recall.RepresentationKind) {
	m.unavailable.WithLabelValues(string(kind)).Inc()
}

func (m *RecallMetrics) Exposition() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
