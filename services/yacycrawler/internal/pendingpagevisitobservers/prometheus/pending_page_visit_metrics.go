package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	labelOutcome = "outcome"
	labelReason  = "reason"

	outcomeReturned       = "returned"
	outcomeTakenByAnother = "taken-by-another"
	outcomeDeferred       = "deferred"
	outcomeRetryScheduled = "retry-scheduled"
	outcomeDisposed       = "disposed"
	outcomeCompleted      = "completed"
)

type PendingPageVisitMetrics struct {
	pageVisitAttempts *prometheusclient.CounterVec
	pagesDisposed     *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PendingPageVisitMetrics {
	metrics := &PendingPageVisitMetrics{
		pageVisitAttempts: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visit_attempts_total",
			Help: "Attempts to visit a page, by the outcome each attempt reached.",
		}, []string{labelOutcome}),
		pagesDisposed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_pages_disposed_total",
			Help: "Pages disposed before a visit could complete, by reason.",
		}, []string{labelReason}),
	}
	registry.MustRegister(metrics.pageVisitAttempts, metrics.pagesDisposed)
	return metrics
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitReturned(
	context.Context, pendingpagevisit.PendingPageVisit, error,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeReturned).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitDroppedAsTakenByAnother(
	context.Context, pendingpagevisit.PendingPageVisit,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeTakenByAnother).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitDeferred(
	context.Context, pendingpagevisit.PendingPageVisit, time.Duration,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeDeferred).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitRetryScheduled(
	context.Context, pendingpagevisit.PendingPageVisit, time.Duration,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeRetryScheduled).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitDisposedPage(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, reason disposal.Reason,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeDisposed).Inc()
	metrics.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitCompleted(
	context.Context,
	pendingpagevisit.PendingPageVisit,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeCompleted).Inc()
}
