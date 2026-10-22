package vaultmetrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	labelWritePhase           = "phase"
	labelWriteOutcome         = "outcome"
	labelWriteCause           = "cause"
	labelWriteCalledOperation = "called_write_operation"

	phaseExecute  = "execute"
	phaseCommit   = "commit"
	phaseRollback = "rollback"

	outcomeCommitted = "committed"
	outcomeAborted   = "aborted"
	outcomeRefused   = "refused"
)

type TransactionMetrics struct {
	beginSeconds  prometheus.Histogram
	beginRefusals *prometheus.CounterVec
	durations     *prometheus.HistogramVec
	transactions  *prometheus.CounterVec
	readsInFlight prometheus.Gauge
}

func NewTransactionMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
) *TransactionMetrics {
	metrics := &TransactionMetrics{
		beginSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    serviceMetricNamespace + "_vault_write_transaction_begin_seconds",
			Help:    "Time spent waiting for a vault write transaction to begin, in seconds.",
			Buckets: prometheus.ExponentialBucketsRange(10e-6, 10, 16),
		}),
		beginRefusals: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceMetricNamespace + "_vault_write_transaction_begin_refusals_total",
				Help: "Vault write transactions the storage engine refused before opening, by cause.",
			},
			[]string{labelWriteCause},
		),
		durations: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    serviceMetricNamespace + "_vault_write_transaction_duration_seconds",
				Help:    "Duration of each phase of an open vault write transaction, in seconds.",
				Buckets: prometheus.ExponentialBucketsRange(10e-6, 10, 16),
			},
			[]string{labelWritePhase},
		),
		transactions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: serviceMetricNamespace + "_vault_write_transactions_total",
				Help: "Vault write transactions that opened, by how they ended.",
			},
			[]string{labelWriteOutcome, labelWriteCause, labelWriteCalledOperation},
		),
		readsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: serviceMetricNamespace + "_vault_reads_in_flight",
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

func (m *TransactionMetrics) ObserveWriteBegan(elapsed time.Duration) {
	m.beginSeconds.Observe(elapsed.Seconds())
}

func (m *TransactionMetrics) ObserveWriteBeginRefused(cause vault.WriteRefusalCause) {
	m.beginRefusals.WithLabelValues(string(cause)).Inc()
}

func (m *TransactionMetrics) ObserveWriteCommitted(
	executed, committed time.Duration,
	calledWriteOperation bool,
) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseCommit).Observe(committed.Seconds())
	m.transactions.WithLabelValues(
		outcomeCommitted,
		"",
		strconv.FormatBool(calledWriteOperation),
	).Inc()
}

func (m *TransactionMetrics) ObserveWriteAborted(executed, rolledBack time.Duration) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseRollback).Observe(rolledBack.Seconds())
	m.transactions.WithLabelValues(outcomeAborted, "", "").Inc()
}

func (m *TransactionMetrics) ObserveWriteCommitRefused(
	executed, rolledBack time.Duration,
	cause vault.WriteRefusalCause,
) {
	m.durations.WithLabelValues(phaseExecute).Observe(executed.Seconds())
	m.durations.WithLabelValues(phaseRollback).Observe(rolledBack.Seconds())
	m.transactions.WithLabelValues(outcomeRefused, string(cause), "").Inc()
}

func (m *TransactionMetrics) ObserveReadBegan() {
	m.readsInFlight.Inc()
}

func (m *TransactionMetrics) ObserveReadEnded() {
	m.readsInFlight.Dec()
}
