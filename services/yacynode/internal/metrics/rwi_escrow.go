package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type RWIEscrowMetrics struct {
	held     prometheus.Counter
	released prometheus.Counter
	expired  prometheus.Counter
	failures prometheus.Counter
}

func NewRWIEscrowMetrics(registry prometheus.Registerer) *RWIEscrowMetrics {
	metrics := &RWIEscrowMetrics{
		held: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rwiescrow_postings_held_total",
			Help: "Inbound postings held because their URL metadata had not arrived.",
		}),
		released: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rwiescrow_postings_released_total",
			Help: "Held postings admitted to the index once their URL metadata arrived.",
		}),
		expired: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rwiescrow_postings_expired_total",
			Help: "Held postings dropped because their URL metadata never arrived.",
		}),
		failures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rwiescrow_expiry_failures_total",
			Help: "Held posting expiry runs that ended in error.",
		}),
	}
	registry.MustRegister(metrics.held, metrics.released, metrics.expired, metrics.failures)

	return metrics
}

func (m *RWIEscrowMetrics) ObserveHeld(postings int) {
	m.held.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveReleased(postings int) {
	m.released.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveExpired(postings int) {
	m.expired.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveExpiryFailure() {
	m.failures.Inc()
}
