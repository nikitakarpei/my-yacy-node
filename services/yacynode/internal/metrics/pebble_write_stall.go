package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const labelWriteStallCause = "cause"

type PebbleWriteStallMetrics struct {
	stalls        *prometheus.CounterVec
	writesDelayed prometheus.Gauge
}

func NewPebbleWriteStallMetrics(registry prometheus.Registerer) *PebbleWriteStallMetrics {
	metrics := &PebbleWriteStallMetrics{
		stalls: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "yacynode_pebble_write_stalls_total",
				Help: "Times the storage engine began to delay writes, by what it waited for.",
			},
			[]string{labelWriteStallCause},
		),
		writesDelayed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "yacynode_pebble_writes_delayed",
			Help: "1 while the storage engine delays writes, 0 while it accepts them.",
		}),
	}
	registry.MustRegister(metrics.stalls, metrics.writesDelayed)

	return metrics
}

func (m *PebbleWriteStallMetrics) ObserveWriteStallBegan(cause pebblevault.WriteStallCause) {
	m.stalls.WithLabelValues(string(cause)).Inc()
	m.writesDelayed.Set(1)
}

func (m *PebbleWriteStallMetrics) ObserveWriteStallEnded() {
	m.writesDelayed.Set(0)
}
