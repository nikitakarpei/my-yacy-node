package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type EscrowCapacity interface {
	Capacity() int
	Count(tx *vault.Txn) (int, error)
}

type RWIEscrowMetrics struct {
	expired  prometheus.Counter
	failures prometheus.Counter
}

func NewRWIEscrowMetrics(registry prometheus.Registerer) *RWIEscrowMetrics {
	metrics := &RWIEscrowMetrics{
		expired: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacynode_rwiescrow_postings_expired_total",
			Help: "Held postings dropped because URL metadata of their version never arrived.",
		}),
		failures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "yacynode_rwiescrow_expiry_failures_total",
			Help: "Held posting expiry runs that ended in error.",
		}),
	}
	registry.MustRegister(
		metrics.expired,
		metrics.failures,
	)

	return metrics
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
	v *vault.Vault,
	escrow EscrowCapacity,
) *RWIEscrowCapacityMetrics {
	capacity := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "yacynode_rwiescrow_capacity_postings",
			Help: "Postings the escrow can hold before it refuses new ones.",
		},
		func() float64 { return float64(escrow.Capacity()) },
	)
	held := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "yacynode_rwiescrow_held_postings",
			Help: "Postings currently held while URL metadata of their version is awaited.",
		},
		func() float64 {
			var postings int
			if err := v.View(context.Background(), func(tx *vault.Txn) error {
				held, err := escrow.Count(tx)
				postings = held

				return err
			}); err != nil {
				return 0
			}

			return float64(postings)
		},
	)
	registry.MustRegister(capacity, held)

	return &RWIEscrowCapacityMetrics{capacity: capacity, held: held}
}
