// Package vaultmetrics counts per bucket the expired records vault forgot and could
// not forget, and shows when the running pass over expired records began and when
// the last pass finished.
package vaultmetrics

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	labelBucket   = "bucket"
	noPassRunning = 0
)

type ExpiryMetrics struct {
	recordsForgotten    *prometheusclient.CounterVec
	recordsNotForgotten *prometheusclient.CounterVec
	passRunningSince    *prometheusclient.GaugeVec
	passFinishedAt      *prometheusclient.GaugeVec
}

func NewExpiryMetrics(
	registry prometheusclient.Registerer,
	serviceMetricNamespace string,
) *ExpiryMetrics {
	metrics := &ExpiryMetrics{
		recordsForgotten: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: serviceMetricNamespace + "_vault_expired_records_forgotten_total",
			Help: "Expired records vault forgot, by bucket.",
		}, []string{labelBucket}),
		recordsNotForgotten: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: serviceMetricNamespace + "_vault_expired_records_not_forgotten_total",
			Help: "Times vault could not forget expired records, by bucket.",
		}, []string{labelBucket}),
		passRunningSince: prometheusclient.NewGaugeVec(prometheusclient.GaugeOpts{
			Name: serviceMetricNamespace + "_vault_expiry_pass_running_since_timestamp_seconds",
			Help: "When the running pass over expired records began, by bucket; 0 when none runs.",
		}, []string{labelBucket}),
		passFinishedAt: prometheusclient.NewGaugeVec(prometheusclient.GaugeOpts{
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
