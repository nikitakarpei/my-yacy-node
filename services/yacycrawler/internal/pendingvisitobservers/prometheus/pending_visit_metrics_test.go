package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	pendingvisitmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisitobservers/prometheus"
)

func TestPendingVisitMetricsCountEachLifecycleFact(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pendingvisitmetricsprometheus.New(registry)
	visit := pendingvisit.PendingVisit{}

	metrics.PendingVisitReturned(context.Background(), visit, errors.New("unavailable"))
	metrics.PendingVisitDroppedBecauseClaimedElsewhere(context.Background(), visit)
	metrics.PendingVisitDeferred(context.Background(), visit, time.Minute)
	metrics.PendingVisitRetryScheduled(context.Background(), visit, time.Minute)
	metrics.PendingVisitDisposedPage(context.Background(), visit, disposal.FetchRejected)
	metrics.PendingVisitCompleted(context.Background(), visit)

	expected := `
# HELP yacycrawler_pages_disposed_total Pages disposed before a visit could complete, by reason.
# TYPE yacycrawler_pages_disposed_total counter
yacycrawler_pages_disposed_total{reason="fetch-rejected"} 1
# HELP yacycrawler_pending_visits_claimed_elsewhere_total Pending visits dropped because another message holds their claim.
# TYPE yacycrawler_pending_visits_claimed_elsewhere_total counter
yacycrawler_pending_visits_claimed_elsewhere_total 1
# HELP yacycrawler_pending_visits_completed_total Pending visits completed.
# TYPE yacycrawler_pending_visits_completed_total counter
yacycrawler_pending_visits_completed_total 1
# HELP yacycrawler_pending_visits_deferred_total Pending visits deferred until a later time.
# TYPE yacycrawler_pending_visits_deferred_total counter
yacycrawler_pending_visits_deferred_total 1
# HELP yacycrawler_pending_visits_retry_scheduled_total Pending visits scheduled for another attempt.
# TYPE yacycrawler_pending_visits_retry_scheduled_total counter
yacycrawler_pending_visits_retry_scheduled_total 1
# HELP yacycrawler_pending_visits_returned_total Pending visits returned for redelivery.
# TYPE yacycrawler_pending_visits_returned_total counter
yacycrawler_pending_visits_returned_total 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
