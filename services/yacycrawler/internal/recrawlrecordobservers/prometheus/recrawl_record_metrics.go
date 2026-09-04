package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RecrawlRecordMetrics struct{ recordsFailed prometheusclient.Counter }

func New(registry prometheusclient.Registerer) *RecrawlRecordMetrics {
	metrics := &RecrawlRecordMetrics{recordsFailed: prometheusclient.NewCounter(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_recrawl_record_failures_total",
			Help: "Recrawl records that could not be stored.",
		},
	)}
	registry.MustRegister(metrics.recordsFailed)
	return metrics
}

func (metrics *RecrawlRecordMetrics) RecrawlRecordFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.recordsFailed.Inc()
}
