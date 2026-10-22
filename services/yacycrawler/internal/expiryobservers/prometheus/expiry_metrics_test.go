package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	expirymetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/expiryobservers/prometheus"
)

func TestExpiryMetricsShowARunningPassUntilItFinishes(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := expirymetricsprometheus.New(registry)
	ctx := context.Background()

	metrics.ExpiryPassBegan(ctx, "crawls", time.Unix(100, 0))
	metrics.ExpiryPassFinished(ctx, "crawls", time.Unix(160, 0))
	metrics.ExpiryPassBegan(ctx, "taken_page_visits", time.Unix(200, 0))
	metrics.ExpiredRecordsForgotten(ctx, "crawls", 3)
	metrics.ExpiredRecordsNotForgotten(ctx, "crawls", errors.New("refused"))

	expected := `
# HELP yacycrawler_expired_records_forgotten_total Expired crawl state records the crawler forgot, by bucket.
# TYPE yacycrawler_expired_records_forgotten_total counter
yacycrawler_expired_records_forgotten_total{bucket="crawls"} 3
# HELP yacycrawler_expired_records_not_forgotten_total Times the crawler could not forget expired crawl state records, by bucket.
# TYPE yacycrawler_expired_records_not_forgotten_total counter
yacycrawler_expired_records_not_forgotten_total{bucket="crawls"} 1
# HELP yacycrawler_expiry_pass_finished_timestamp_seconds When the last pass over expired records finished, by bucket.
# TYPE yacycrawler_expiry_pass_finished_timestamp_seconds gauge
yacycrawler_expiry_pass_finished_timestamp_seconds{bucket="crawls"} 160
# HELP yacycrawler_expiry_pass_running_since_timestamp_seconds When the running pass over expired records began, by bucket; 0 when none runs.
# TYPE yacycrawler_expiry_pass_running_since_timestamp_seconds gauge
yacycrawler_expiry_pass_running_since_timestamp_seconds{bucket="crawls"} 0
yacycrawler_expiry_pass_running_since_timestamp_seconds{bucket="taken_page_visits"} 200
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}
