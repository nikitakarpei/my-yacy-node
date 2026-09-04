package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelUnreadable = "unreadable"

	unreadablePageHTML = "page-html"
)

type PageVisitFailureMetrics struct {
	pageVisitFailures *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PageVisitFailureMetrics {
	metrics := &PageVisitFailureMetrics{
		pageVisitFailures: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visit_failures_total",
			Help: "Page visits left for another attempt, by what the crawler could not read.",
		}, []string{labelUnreadable}),
	}
	registry.MustRegister(metrics.pageVisitFailures)
	return metrics
}

func (metrics *PageVisitFailureMetrics) PageHTMLUnreadable(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.pageVisitFailures.WithLabelValues(unreadablePageHTML).Inc()
}
