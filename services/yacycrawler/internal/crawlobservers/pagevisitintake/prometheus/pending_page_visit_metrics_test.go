package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	pendingpagevisitmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlobservers/pagevisitintake/prometheus"
)

func TestPageVisitAttemptsCountOneOutcomeEach(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pendingpagevisitmetricsprometheus.New(registry)
	visit := pendingpagevisit.PendingPageVisit{}

	metrics.PendingPageVisitReturned(context.Background(), visit, errors.New("unavailable"))
	metrics.PendingPageVisitDropped(
		context.Background(),
		visit,
		yacycrawlcontract.DropReasonNotDue,
	)
	metrics.PendingPageVisitDeferred(context.Background(), visit, time.Minute)
	metrics.PendingPageVisitRetryScheduled(context.Background(), visit, time.Minute)
	metrics.PendingPageVisitCompleted(
		context.Background(),
		visit,
		yacycrawlcontract.ResultFetchRejected,
	)

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
yacycrawler_page_visits_dropped_total{reason="not-due"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacycrawler_page_visit_attempts_total",
		"yacycrawler_page_visits_completed_total",
		"yacycrawler_page_visits_dropped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestPageVisitsLeasedAtOnceAreObservedPerLease(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pendingpagevisitmetricsprometheus.New(registry)

	metrics.PendingPageVisitsLeased(context.Background(), 1)
	metrics.PendingPageVisitsLeased(context.Background(), 3)

	expected := `
# HELP yacycrawler_page_visits_leased_at_once Pending page visits leased at once for the workers free at that moment.
# TYPE yacycrawler_page_visits_leased_at_once histogram
yacycrawler_page_visits_leased_at_once_bucket{le="1"} 1
yacycrawler_page_visits_leased_at_once_bucket{le="2"} 1
yacycrawler_page_visits_leased_at_once_bucket{le="4"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="8"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="16"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="32"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="64"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="128"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="256"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="512"} 2
yacycrawler_page_visits_leased_at_once_bucket{le="+Inf"} 2
yacycrawler_page_visits_leased_at_once_sum 4
yacycrawler_page_visits_leased_at_once_count 2
`
	if err := testutil.GatherAndCompare(
		registry, strings.NewReader(expected), "yacycrawler_page_visits_leased_at_once",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
