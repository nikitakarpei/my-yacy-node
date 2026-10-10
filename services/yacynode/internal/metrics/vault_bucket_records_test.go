package metrics_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type stubVaultBuckets struct {
	recordCounts map[vault.Name]int
	err          error
}

func (s stubVaultBuckets) RecordCountsByBucket(
	context.Context,
) (map[vault.Name]int, error) {
	return s.recordCounts, s.err
}

func TestVaultBucketRecordsReportsRecordCounts(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewVaultBucketRecordMetrics(registry, stubVaultBuckets{
		recordCounts: map[vault.Name]int{"rwi": 7, "urlmeta": 0},
	})

	expected := `
# HELP yacynode_vault_bucket_records Records currently stored in each vault bucket of records.
# TYPE yacynode_vault_bucket_records gauge
yacynode_vault_bucket_records{bucket="rwi"} 7
yacynode_vault_bucket_records{bucket="urlmeta"} 0
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultBucketRecordsOmitsRecordCountsOnError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewVaultBucketRecordMetrics(
		registry,
		stubVaultBuckets{err: errors.New("unavailable")},
	)

	if got := testutil.CollectAndCount(registry, "yacynode_vault_bucket_records"); got != 0 {
		t.Errorf("yacynode_vault_bucket_records samples = %d, want none on error", got)
	}
}
