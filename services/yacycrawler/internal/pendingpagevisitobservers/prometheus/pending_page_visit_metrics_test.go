package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	pendingpagevisitmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisitobservers/prometheus"
)

func TestPageVisitAttemptsCountOneOutcomeEach(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pendingpagevisitmetricsprometheus.New(registry)
	visit := pendingpagevisit.PendingPageVisit{}

	metrics.PendingPageVisitReturned(context.Background(), visit, errors.New("unavailable"))
	metrics.PendingPageVisitDropped(context.Background(), visit, pagevisitdrop.TakenByAnother)
	metrics.PendingPageVisitDeferred(context.Background(), visit, time.Minute)
	metrics.PendingPageVisitRetryScheduled(context.Background(), visit, time.Minute)
	metrics.PendingPageVisitCompleted(context.Background(), visit, pagevisitresult.FetchRejected)

	expected := `
# HELP yacycrawler_page_visit_attempts_total Attempts to visit a page, by the outcome each attempt reached.
# TYPE yacycrawler_page_visit_attempts_total counter
yacycrawler_page_visit_attempts_total{outcome="completed"} 1
yacycrawler_page_visit_attempts_total{outcome="deferred"} 1
yacycrawler_page_visit_attempts_total{outcome="dropped"} 1
yacycrawler_page_visit_attempts_total{outcome="retry-scheduled"} 1
yacycrawler_page_visit_attempts_total{outcome="returned"} 1
# HELP yacycrawler_page_visits_completed_total Page visits completed, by result.
# TYPE yacycrawler_page_visits_completed_total counter
yacycrawler_page_visits_completed_total{result="fetch-rejected"} 1
# HELP yacycrawler_page_visits_dropped_total Pages the crawler stopped trying to visit, by reason.
# TYPE yacycrawler_page_visits_dropped_total counter
yacycrawler_page_visits_dropped_total{reason="taken-by-another"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
