package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitRecordMetrics struct{ pageVisitsNotRecorded prometheusclient.Counter }

func New(registry prometheusclient.Registerer) *PageVisitRecordMetrics {
	metrics := &PageVisitRecordMetrics{pageVisitsNotRecorded: prometheusclient.NewCounter(
		prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visits_not_recorded_total",
			Help: "Page visits the crawler could not record.",
		},
	)}
	registry.MustRegister(metrics.pageVisitsNotRecorded)
	return metrics
}

func (metrics *PageVisitRecordMetrics) PageVisitNotRecorded(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.pageVisitsNotRecorded.Inc()
}
