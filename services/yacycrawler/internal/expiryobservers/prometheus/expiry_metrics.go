// Package prometheus counts per bucket the expired crawl state records the crawler
// forgot and could not forget, and shows when the running pass over expired records
// began and when the last pass finished.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const noPassRunning = 0

type ExpiryMetrics struct {
	recordsForgotten    *prometheusclient.CounterVec
	recordsNotForgotten *prometheusclient.CounterVec
	passRunningSince    *prometheusclient.GaugeVec
	passFinishedAt      *prometheusclient.GaugeVec
}

func New(registry prometheusclient.Registerer) *ExpiryMetrics {
	metrics := &ExpiryMetrics{
		recordsForgotten: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_expired_records_forgotten_total",
			Help: "Expired crawl state records the crawler forgot, by bucket.",
		}, []string{"bucket"}),
		recordsNotForgotten: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_expired_records_not_forgotten_total",
			Help: "Times the crawler could not forget expired crawl state records, by bucket.",
		}, []string{"bucket"}),
		passRunningSince: prometheusclient.NewGaugeVec(prometheusclient.GaugeOpts{
			Name: "yacycrawler_expiry_pass_running_since_timestamp_seconds",
			Help: "When the running pass over expired records began, by bucket; 0 when none runs.",
		}, []string{"bucket"}),
		passFinishedAt: prometheusclient.NewGaugeVec(prometheusclient.GaugeOpts{
			Name: "yacycrawler_expiry_pass_finished_timestamp_seconds",
			Help: "When the last pass over expired records finished, by bucket.",
		}, []string{"bucket"}),
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
