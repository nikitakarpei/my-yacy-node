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

func TestPageVisitAttemptsCountOneOutcomeEach(t *testing.T) {
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
# HELP yacycrawler_page_visit_attempts_total Attempts to visit a page, by the outcome each attempt reached.
# TYPE yacycrawler_page_visit_attempts_total counter
yacycrawler_page_visit_attempts_total{outcome="claimed-elsewhere"} 1
yacycrawler_page_visit_attempts_total{outcome="completed"} 1
yacycrawler_page_visit_attempts_total{outcome="deferred"} 1
yacycrawler_page_visit_attempts_total{outcome="disposed"} 1
yacycrawler_page_visit_attempts_total{outcome="retry-scheduled"} 1
yacycrawler_page_visit_attempts_total{outcome="returned"} 1
# HELP yacycrawler_pages_disposed_total Pages disposed before a visit could complete, by reason.
# TYPE yacycrawler_pages_disposed_total counter
yacycrawler_pages_disposed_total{reason="fetch-rejected"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
