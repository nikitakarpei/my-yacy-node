package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	labelWritePhase   = "phase"
	labelWriteOutcome = "outcome"
	labelWriteCause   = "cause"

	phaseExecute  = "execute"
	phaseCommit   = "commit"
	phaseRollback = "rollback"

	outcomeCommitted = "committed"
	outcomeAborted   = "aborted"
	outcomeRefused   = "refused"
)

type VaultTransactionMetrics struct {
	beginSeconds  prometheus.Histogram
	beginRefusals *prometheus.CounterVec
	durations     *prometheus.HistogramVec
	transactions  *prometheus.CounterVec
	readsInFlight prometheus.Gauge
}

func NewVaultTransactionMetrics(registry prometheus.Registerer) *VaultTransactionMetrics {
	metrics := &VaultTransactionMetrics{
		beginSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "vault_write_transaction_begin_seconds",
			Help:    "Time spent waiting for a vault write transaction to begin, in seconds.",
			Buckets: prometheus.ExponentialBucketsRange(10e-6, 10, 16),
		}),
		beginRefusals: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vault_write_transaction_begin_refusals_total",
				Help: "Vault write transactions the storage engine refused before opening, by cause.",
			},
			[]string{labelWriteCause},
		),
		durations: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vault_write_transaction_duration_seconds",
				Help:    "Duration of each phase of an open vault write transaction, in seconds.",
				Buckets: prometheus.ExponentialBucketsRange(10e-6, 10, 16),
			},
			[]string{labelWritePhase},
		),
		transactions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vault_write_transactions_total",
				Help: "Vault write transactions that opened, by how they ended.",
			},
			[]string{labelWriteOutcome, labelWriteCause},
		),
		readsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "vault_reads_in_flight",
			Help: "Reads currently open through the vault.",
		}),
	}
	registry.MustRegister(
		metrics.beginSeconds,
		metrics.beginRefusals,
		metrics.durations,
		metrics.transactions,
		metrics.readsInFlight,
	)

	return metrics
}

func (m *VaultTransactionMetrics) ObserveWriteBegan(elapsed time.Duration) {
	m.beginSeconds.Observe(elapsed.Seconds())
}

func (m *VaultTransactionMetrics) ObserveWriteBeginRefused(cause vault.WriteRefusalCause) {
	m.beginRefusals.WithLabelValues(string(cause)).Inc()
}

func (m *VaultTransactionMetrics) ObserveWriteCommitted(executed, committed time.Duration) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseCommit).Observe(committed.Seconds())
	m.transactions.WithLabelValues(outcomeCommitted, "").Inc()
}

func (m *VaultTransactionMetrics) ObserveWriteAborted(executed, rolledBack time.Duration) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseRollback).Observe(rolledBack.Seconds())
	m.transactions.WithLabelValues(outcomeAborted, "").Inc()
}

func (m *VaultTransactionMetrics) ObserveWriteCommitRefused(
	executed, rolledBack time.Duration,
	cause vault.WriteRefusalCause,
) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseRollback).Observe(rolledBack.Seconds())
	m.transactions.WithLabelValues(outcomeRefused, string(cause)).Inc()
}

func (m *VaultTransactionMetrics) ObserveReadBegan() {
	m.readsInFlight.Inc()
}

func (m *VaultTransactionMetrics) ObserveReadEnded() {
	m.readsInFlight.Dec()
}
