package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	pagefetchmetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefetchobservers/prometheus"
)

func TestPageFetchMetricsCountConcreteFetchFacts(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := pagefetchmetricsprometheus.New(registry)
	pageURL := canonicalurl.CanonicalURL{}
	fetchDuration := time.Second

	metrics.PageFetchSucceeded(context.Background(), pageURL, fetchDuration)
	metrics.PageFetchNotModified(context.Background(), pageURL, fetchDuration)
	metrics.PageFetchAccessRefused(context.Background(), pageURL, fetchDuration)
	metrics.PageFetchDeferred(context.Background(), pageURL, fetchDuration, time.Minute)
	metrics.PageFetchRejected(context.Background(), pageURL, fetchDuration)
	metrics.PageFetchRedirected(context.Background(), pageURL, pageURL, fetchDuration)
	metrics.PageFetchRedirectTargetInvalid(
		context.Background(),
		pageURL,
		fetchDuration,
		errors.New("bad url"),
	)
	metrics.PageFetchRefusedOversizedPage(context.Background(), pageURL, fetchDuration)
	metrics.PageFetchFailed(context.Background(), pageURL, fetchDuration, errors.New("unavailable"))
	metrics.PageFetchCanceled(context.Background(), pageURL, fetchDuration)

	expected := `
# HELP yacycrawler_page_fetches_processed_total Page fetches processed, by outcome.
# TYPE yacycrawler_page_fetches_processed_total counter
yacycrawler_page_fetches_processed_total{outcome="access_refused"} 1
yacycrawler_page_fetches_processed_total{outcome="canceled"} 1
yacycrawler_page_fetches_processed_total{outcome="deferred"} 1
yacycrawler_page_fetches_processed_total{outcome="failed"} 1
yacycrawler_page_fetches_processed_total{outcome="not_modified"} 1
yacycrawler_page_fetches_processed_total{outcome="oversized"} 1
yacycrawler_page_fetches_processed_total{outcome="redirect_target_invalid"} 1
yacycrawler_page_fetches_processed_total{outcome="redirected"} 1
yacycrawler_page_fetches_processed_total{outcome="rejected"} 1
yacycrawler_page_fetches_processed_total{outcome="succeeded"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacycrawler_page_fetches_processed_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
	if got := testutil.CollectAndCount(
		registry,
		"yacycrawler_page_fetch_duration_seconds",
	); got != 1 {
		t.Errorf("fetch duration metrics = %d, want 1", got)
	}
}
