package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type EscrowCapacity interface {
	Capacity() int
	Count(context.Context) (int, error)
}

type RWIEscrowMetrics struct {
	held     prometheus.Counter
	released prometheus.Counter
	refused  prometheus.Counter
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
		refused: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rwiescrow_postings_refused_total",
			Help: "Inbound postings dropped because the escrow was at capacity.",
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
	registry.MustRegister(
		metrics.held,
		metrics.released,
		metrics.refused,
		metrics.expired,
		metrics.failures,
	)

	return metrics
}

func (m *RWIEscrowMetrics) ObserveHeld(postings int) {
	m.held.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveReleased(postings int) {
	m.released.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveRefused(postings int) {
	m.refused.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveExpired(postings int) {
	m.expired.Add(float64(postings))
}

func (m *RWIEscrowMetrics) ObserveExpiryFailure() {
	m.failures.Inc()
}

type RWIEscrowCapacityMetrics struct {
	capacity prometheus.GaugeFunc
	held     prometheus.GaugeFunc
}

func NewRWIEscrowCapacityMetrics(
	registry prometheus.Registerer,
	escrow EscrowCapacity,
) *RWIEscrowCapacityMetrics {
	capacity := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "rwiescrow_capacity_postings",
			Help: "Postings the escrow can hold before it refuses new ones.",
		},
		func() float64 { return float64(escrow.Capacity()) },
	)
	held := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "rwiescrow_held_postings",
			Help: "Postings currently held while their URL metadata is awaited.",
		},
		func() float64 {
			postings, err := escrow.Count(context.Background())
			if err != nil {
				return 0
			}

			return float64(postings)
		},
	)
	registry.MustRegister(capacity, held)

	return &RWIEscrowCapacityMetrics{capacity: capacity, held: held}
}
