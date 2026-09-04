package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

const (
	labelReason = "reason"
)

type PendingVisitMetrics struct {
	pendingVisitsReturned         prometheusclient.Counter
	pendingVisitsClaimedElsewhere prometheusclient.Counter
	pendingVisitsDeferred         prometheusclient.Counter
	pendingVisitsRetryScheduled   prometheusclient.Counter
	pendingVisitsCompleted        prometheusclient.Counter
	pagesDisposed                 *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *PendingVisitMetrics {
	metrics := &PendingVisitMetrics{
		pendingVisitsReturned: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_returned_total",
			Help: "Pending visits returned for redelivery.",
		}),
		pendingVisitsClaimedElsewhere: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_claimed_elsewhere_total",
			Help: "Pending visits dropped because another message holds their claim.",
		}),
		pendingVisitsDeferred: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_deferred_total",
			Help: "Pending visits deferred until a later time.",
		}),
		pendingVisitsRetryScheduled: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_retry_scheduled_total",
			Help: "Pending visits scheduled for another attempt.",
		}),
		pendingVisitsCompleted: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_pending_visits_completed_total",
			Help: "Pending visits completed.",
		}),
		pagesDisposed: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_pages_disposed_total",
			Help: "Pages disposed before a visit could complete, by reason.",
		}, []string{labelReason}),
	}
	registry.MustRegister(
		metrics.pendingVisitsReturned,
		metrics.pendingVisitsClaimedElsewhere,
		metrics.pendingVisitsDeferred,
		metrics.pendingVisitsRetryScheduled,
		metrics.pendingVisitsCompleted,
		metrics.pagesDisposed,
	)
	return metrics
}

func (metrics *PendingVisitMetrics) PendingVisitReturned(
	context.Context, pendingvisit.PendingVisit, error,
) {
	metrics.pendingVisitsReturned.Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDroppedBecauseClaimedElsewhere(
	context.Context, pendingvisit.PendingVisit,
) {
	metrics.pendingVisitsClaimedElsewhere.Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDeferred(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.pendingVisitsDeferred.Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitRetryScheduled(
	context.Context, pendingvisit.PendingVisit, time.Duration,
) {
	metrics.pendingVisitsRetryScheduled.Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitDisposedPage(
	_ context.Context, _ pendingvisit.PendingVisit, reason disposal.Reason,
) {
	metrics.pagesDisposed.WithLabelValues(string(reason)).Inc()
}

func (metrics *PendingVisitMetrics) PendingVisitCompleted(
	context.Context,
	pendingvisit.PendingVisit,
) {
	metrics.pendingVisitsCompleted.Inc()
}
