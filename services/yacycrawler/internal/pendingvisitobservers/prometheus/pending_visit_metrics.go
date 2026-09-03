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
)

type PendingVisitMetrics struct {
	visitsProcessed *prometheusclient.CounterVec
	pagesDisposed   *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PendingVisitMetrics {
	metrics := &PendingVisitMetrics{
		visitsProcessed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_processed_total",
			Help: "Pending visits processed, by outcome.",
		}, []string{labelOutcome}),
		pagesDisposed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_pages_disposed_total",
			Help: "Pages disposed before a visit could complete, by reason.",
		}, []string{labelReason}),
	}
	registry.MustRegister(metrics.visitsProcessed, metrics.pagesDisposed)
	return metrics
}

func (metrics *PendingVisitMetrics) PendingVisitReturned(
	context.Context, pendingvisit.PendingVisit, error,
) {
	metrics.visitsProcessed.WithLabelValues("returned").Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDroppedBecauseClaimedElsewhere(
	context.Context, pendingvisit.PendingVisit,
) {
	metrics.visitsProcessed.WithLabelValues("claimed_elsewhere").Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDeferred(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.visitsProcessed.WithLabelValues("deferred").Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitRetryScheduled(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.visitsProcessed.WithLabelValues("retry_scheduled").Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDisposedPage(
	_ context.Context, _ pendingvisit.PendingVisit, reason disposal.Reason,
) {
	metrics.visitsProcessed.WithLabelValues("disposed").Inc()
	metrics.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitCompleted(
	context.Context,
	pendingvisit.PendingVisit,
) {
	metrics.visitsProcessed.WithLabelValues("completed").Inc()
}
