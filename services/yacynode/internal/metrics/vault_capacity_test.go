package metrics_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type stubVaultCapacity struct {
	quota int64
	used  int64
	err   error
}

func (s stubVaultCapacity) QuotaBytes() int64 { return s.quota }

func (s stubVaultCapacity) UsedBytes(context.Context) (int64, error) { return s.used, s.err }

func TestVaultCapacityReportsLevels(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewVaultCapacityMetrics(registry, stubVaultCapacity{quota: 1024, used: 256})

	expected := `
# HELP yacynode_vault_quota_bytes Configured vault quota in bytes.
# TYPE yacynode_vault_quota_bytes gauge
yacynode_vault_quota_bytes 1024
# HELP yacynode_vault_used_bytes Vault space currently used in bytes.
# TYPE yacynode_vault_used_bytes gauge
yacynode_vault_used_bytes 256
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultCapacityOmitsUsedBytesOnError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewVaultCapacityMetrics(
		registry,
		stubVaultCapacity{quota: 1024, used: 256, err: errors.New("unavailable")},
	)

	if got := testutil.CollectAndCount(registry, "yacynode_vault_used_bytes"); got != 0 {
		t.Errorf("yacynode_vault_used_bytes samples = %d, want none on error", got)
	}
	if got := testutil.CollectAndCount(registry, "yacynode_vault_quota_bytes"); got != 1 {
		t.Errorf("yacynode_vault_quota_bytes samples = %d, want 1", got)
	}
}
