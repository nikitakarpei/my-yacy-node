package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome            = "outcome"
	outcomeRedirected       = "redirected"
	outcomeRejected         = "rejected"
	outcomeMethodNotAllowed = "method_not_allowed"
)

type VisitedPageMetrics struct {
	requestsProcessed *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *VisitedPageMetrics {
	metrics := &VisitedPageMetrics{
		requestsProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "visitcrawl_visited_page_requests_processed_total",
			Help: "Visited-page requests processed, by outcome.",
		}, []string{labelOutcome}),
	}
	registry.MustRegister(metrics.requestsProcessed)
	return metrics
}

func (metrics *VisitedPageMetrics) VisitedPageRedirected(context.Context, string) {
	metrics.requestsProcessed.WithLabelValues(outcomeRedirected).Inc()
}

func (metrics *VisitedPageMetrics) VisitedPageRejected(context.Context, error) {
	metrics.requestsProcessed.WithLabelValues(outcomeRejected).Inc()
}

func (metrics *VisitedPageMetrics) VisitedPageMethodRefused(context.Context, string) {
	metrics.requestsProcessed.WithLabelValues(outcomeMethodNotAllowed).Inc()
}
