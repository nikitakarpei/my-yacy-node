// Package vaultmetrics exposes vault transactions, capacity, records per bucket,
// member sets and the removal of expired records as Prometheus metrics under the
// namespace of the service that wires them.
package vaultmetrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const noPassRunning = 0

type ExpiryMetrics struct {
	recordsForgotten    *prometheus.CounterVec
	recordsNotForgotten *prometheus.CounterVec
	passRunningSince    *prometheus.GaugeVec
	passFinishedAt      *prometheus.GaugeVec
}

func NewExpiryMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
) *ExpiryMetrics {
	metrics := &ExpiryMetrics{
		recordsForgotten: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: serviceMetricNamespace + "_vault_expired_records_forgotten_total",
			Help: "Expired records vault forgot, by bucket.",
		}, []string{labelBucket}),
		recordsNotForgotten: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: serviceMetricNamespace + "_vault_expired_records_not_forgotten_total",
			Help: "Times vault could not forget expired records, by bucket.",
		}, []string{labelBucket}),
		passRunningSince: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: serviceMetricNamespace + "_vault_expiry_pass_running_since_timestamp_seconds",
			Help: "When the running pass over expired records began, by bucket; 0 when none runs.",
		}, []string{labelBucket}),
		passFinishedAt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: serviceMetricNamespace + "_vault_expiry_pass_finished_timestamp_seconds",
			Help: "When the last pass over expired records finished, by bucket.",
		}, []string{labelBucket}),
	}
	registry.MustRegister(
		metrics.recordsForgotten,
		metrics.recordsNotForgotten,
		metrics.passRunningSince,
		metrics.passFinishedAt,
	)

	return metrics
}

func (metrics *ExpiryMetrics) ExpiryPassBegan(_ context.Context, bucket vault.Name, at time.Time) {
	metrics.passRunningSince.WithLabelValues(string(bucket)).Set(secondsOf(at))
}

func secondsOf(at time.Time) float64 {
	return float64(at.UnixNano()) / float64(time.Second)
}

func (metrics *ExpiryMetrics) ExpiryPassFinished(
	_ context.Context,
	bucket vault.Name,
	at time.Time,
) {
	metrics.passRunningSince.WithLabelValues(string(bucket)).Set(noPassRunning)
	metrics.passFinishedAt.WithLabelValues(string(bucket)).Set(secondsOf(at))
}

func (metrics *ExpiryMetrics) ExpiredRecordsForgotten(
	_ context.Context,
	bucket vault.Name,
	amount int,
) {
	metrics.recordsForgotten.WithLabelValues(string(bucket)).Add(float64(amount))
}

func (metrics *ExpiryMetrics) ExpiredRecordsNotForgotten(
	_ context.Context,
	bucket vault.Name,
	_ error,
) {
	metrics.recordsNotForgotten.WithLabelValues(string(bucket)).Inc()
}
