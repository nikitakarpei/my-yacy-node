package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	labelOutcome = "outcome"
	labelReason  = "reason"
	labelResult  = "result"

	outcomeReturned       = "returned"
	outcomeDeferred       = "deferred"
	outcomeRetryScheduled = "retry-scheduled"
	outcomeDropped        = "dropped"
	outcomeCompleted      = "completed"
)

type PendingPageVisitMetrics struct {
	pageVisitAttempts   *prometheusclient.CounterVec
	droppedPageVisits   *prometheusclient.CounterVec
	completedPageVisits *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PendingPageVisitMetrics {
	metrics := &PendingPageVisitMetrics{
		pageVisitAttempts: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visit_attempts_total",
			Help: "Attempts to visit a page, by the outcome each attempt reached.",
		}, []string{labelOutcome}),
		droppedPageVisits: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visits_dropped_total",
			Help: "Pages the crawler stopped trying to visit, by reason.",
		}, []string{labelReason}),
		completedPageVisits: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_page_visits_completed_total",
			Help: "Page visits completed, by result.",
		}, []string{labelResult}),
	}
	registry.MustRegister(
		metrics.pageVisitAttempts, metrics.droppedPageVisits, metrics.completedPageVisits,
	)
	return metrics
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitReturned(
	context.Context, pendingpagevisit.PendingPageVisit, error,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeReturned).Inc()
}

func (metrics *PendingPageVisitMetrics) PendingPageVisitDropped(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, reason pagevisitdrop.Reason,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeDropped).Inc()
	metrics.droppedPageVisits.WithLabelValues(string(reason)).Inc()
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

func (metrics *PendingPageVisitMetrics) PendingPageVisitCompleted(
	_ context.Context, _ pendingpagevisit.PendingPageVisit, result pagevisitresult.Result,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeCompleted).Inc()
	metrics.completedPageVisits.WithLabelValues(string(result)).Inc()
}
