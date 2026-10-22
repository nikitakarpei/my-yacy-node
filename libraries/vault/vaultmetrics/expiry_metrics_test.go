package vaultmetrics_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultmetrics"
)

func TestExpiryMetricsShowARunningPassUntilItFinishes(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := vaultmetrics.NewExpiryMetrics(registry, "service")
	ctx := context.Background()

	metrics.ExpiryPassBegan(ctx, "crawls", time.Unix(100, 0))
	metrics.ExpiryPassFinished(ctx, "crawls", time.Unix(160, 0))
	metrics.ExpiryPassBegan(ctx, "taken_page_visits", time.Unix(200, 0))
	metrics.ExpiredRecordsForgotten(ctx, "crawls", 3)
	metrics.ExpiredRecordsNotForgotten(ctx, "crawls", errors.New("refused"))

	expected := `
# HELP service_vault_expired_records_forgotten_total Expired records vault forgot, by bucket.
# TYPE service_vault_expired_records_forgotten_total counter
service_vault_expired_records_forgotten_total{bucket="crawls"} 3
# HELP service_vault_expired_records_not_forgotten_total Times vault could not forget expired records, by bucket.
# TYPE service_vault_expired_records_not_forgotten_total counter
service_vault_expired_records_not_forgotten_total{bucket="crawls"} 1
# HELP service_vault_expiry_pass_finished_timestamp_seconds When the last pass over expired records finished, by bucket.
# TYPE service_vault_expiry_pass_finished_timestamp_seconds gauge
service_vault_expiry_pass_finished_timestamp_seconds{bucket="crawls"} 160
# HELP service_vault_expiry_pass_running_since_timestamp_seconds When the running pass over expired records began, by bucket; 0 when none runs.
# TYPE service_vault_expiry_pass_running_since_timestamp_seconds gauge
service_vault_expiry_pass_running_since_timestamp_seconds{bucket="crawls"} 0
service_vault_expiry_pass_running_since_timestamp_seconds{bucket="taken_page_visits"} 200
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}
