package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitRecordMetrics struct {
	pageVisitsNotRecorded prometheusclient.Counter
	lastPageVisitsNotRead prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *PageVisitRecordMetrics {
	metrics := &PageVisitRecordMetrics{
		pageVisitsNotRecorded: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visits_not_recorded_total",
			Help: "Page visits the crawler could not record.",
		}),
		lastPageVisitsNotRead: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_last_page_visits_not_read_total",
			Help: "Last page visits the crawler could not read, so it fetched the page again.",
		}),
	}
	registry.MustRegister(metrics.pageVisitsNotRecorded, metrics.lastPageVisitsNotRead)
	return metrics
}

func (metrics *PageVisitRecordMetrics) PageVisitNotRecorded(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.pageVisitsNotRecorded.Inc()
}

func (metrics *PageVisitRecordMetrics) LastPageVisitNotRead(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.lastPageVisitsNotRead.Inc()
}
