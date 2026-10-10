package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestWriteConflictsAreCountedByBucketAndKeyKind(t *testing.T) {
	registry := prometheus.NewRegistry()
	conflicts := metrics.NewPebbleWriteConflictMetrics(registry)

	conflicts.ObserveWriteConflicted("rwi_escrow", pebblevault.KindTallies)
	conflicts.ObserveWriteConflicted("rwi_escrow", pebblevault.KindTallies)
	conflicts.ObserveWriteConflicted("urlmeta", pebblevault.KindRecords)

	expected := `
# HELP yacynode_pebble_write_conflicts_total Write transactions that read a key another write committed after they began, by the bucket and kind of that key. Each one runs again.
# TYPE yacynode_pebble_write_conflicts_total counter
yacynode_pebble_write_conflicts_total{bucket="rwi_escrow",kind="tallies"} 2
yacynode_pebble_write_conflicts_total{bucket="urlmeta",kind="records"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_write_conflicts_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestExclusiveWritesAreCounted(t *testing.T) {
	registry := prometheus.NewRegistry()
	conflicts := metrics.NewPebbleWriteConflictMetrics(registry)

	conflicts.ObserveExclusiveWrite()

	expected := `
# HELP yacynode_pebble_exclusive_writes_total Write transactions that conflicted too often and ran while other writes waited to commit.
# TYPE yacynode_pebble_exclusive_writes_total counter
yacynode_pebble_exclusive_writes_total 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_exclusive_writes_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
