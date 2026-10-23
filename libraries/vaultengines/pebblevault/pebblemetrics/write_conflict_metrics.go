package pebblemetrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	labelConflictBucket  = "bucket"
	labelConflictKeyKind = "kind"
)

type WriteConflictMetrics struct {
	conflicts       *prometheus.CounterVec
	exclusiveWrites prometheus.Counter
}

func NewWriteConflictMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
) *WriteConflictMetrics {
	metrics := &WriteConflictMetrics{
		conflicts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceMetricNamespace + "_pebble_write_conflicts_total",
				Help: "Write transactions that read a key another write committed after they began, " +
					"by the bucket and kind of that key. Each one runs again.",
			},
			[]string{labelConflictBucket, labelConflictKeyKind},
		),
		exclusiveWrites: prometheus.NewCounter(prometheus.CounterOpts{
			Name: serviceMetricNamespace + "_pebble_exclusive_writes_total",
			Help: "Write transactions that conflicted too often and ran while other writes waited to commit.",
		}),
	}
	registry.MustRegister(metrics.conflicts, metrics.exclusiveWrites)

	return metrics
}

func (m *WriteConflictMetrics) ObserveWriteConflicted(
	bucket vault.Name,
	kind pebblevault.KeyKind,
) {
	m.conflicts.WithLabelValues(string(bucket), string(kind)).Inc()
}

func (m *WriteConflictMetrics) ObserveExclusiveWrite() {
	m.exclusiveWrites.Inc()
}
