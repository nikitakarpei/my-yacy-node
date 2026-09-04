package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

const (
	labelOutcome = "outcome"
	labelReason  = "reason"

	outcomeReturned         = "returned"
	outcomeClaimedElsewhere = "claimed-elsewhere"
	outcomeDeferred         = "deferred"
	outcomeRetryScheduled   = "retry-scheduled"
	outcomeDisposed         = "disposed"
	outcomeCompleted        = "completed"
)

type PendingVisitMetrics struct {
	pageVisitAttempts *prometheusclient.CounterVec
	pagesDisposed     *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PendingVisitMetrics {
	metrics := &PendingVisitMetrics{
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

func (metrics *PendingVisitMetrics) PendingVisitReturned(
	context.Context, pendingvisit.PendingVisit, error,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeReturned).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDroppedBecauseClaimedElsewhere(
	context.Context, pendingvisit.PendingVisit,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeClaimedElsewhere).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDeferred(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeDeferred).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitRetryScheduled(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeRetryScheduled).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDisposedPage(
	_ context.Context, _ pendingvisit.PendingVisit, reason disposal.Reason,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeDisposed).Inc()
	metrics.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitCompleted(
	context.Context,
	pendingvisit.PendingVisit,
) {
	metrics.pageVisitAttempts.WithLabelValues(outcomeCompleted).Inc()
}
