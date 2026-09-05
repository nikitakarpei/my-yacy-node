package metrics_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestWriteStallsAreCountedByCause(t *testing.T) {
	registry := prometheus.NewRegistry()
	stalls := metrics.NewPebbleWriteStallMetrics(registry)

	stalls.ObserveWriteStallBegan("memtables_awaiting_flush")
	stalls.ObserveWriteStallEnded()
	stalls.ObserveWriteStallBegan("memtables_awaiting_flush")
	stalls.ObserveWriteStallEnded()
	stalls.ObserveWriteStallBegan("level_zero_files_awaiting_compaction")
	stalls.ObserveWriteStallEnded()

	expected := `
# HELP yacynode_pebble_write_stalls_total Times the storage engine began to delay writes, by what it waited for.
# TYPE yacynode_pebble_write_stalls_total counter
yacynode_pebble_write_stalls_total{cause="level_zero_files_awaiting_compaction"} 1
yacynode_pebble_write_stalls_total{cause="memtables_awaiting_flush"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_write_stalls_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDelayedWritesAreVisibleUntilTheStallEnds(t *testing.T) {
	registry := prometheus.NewRegistry()
	stalls := metrics.NewPebbleWriteStallMetrics(registry)

	expectDelayedWrites(t, registry, 0)

	stalls.ObserveWriteStallBegan("memtables_awaiting_flush")
	expectDelayedWrites(t, registry, 1)

	stalls.ObserveWriteStallEnded()
	expectDelayedWrites(t, registry, 0)
}

func expectDelayedWrites(t *testing.T, registry *prometheus.Registry, delayed int) {
	t.Helper()

	expected := fmt.Sprintf(`
# HELP yacynode_pebble_writes_delayed 1 while the storage engine delays writes, 0 while it accepts them.
# TYPE yacynode_pebble_writes_delayed gauge
yacynode_pebble_writes_delayed %d
`, delayed)
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_writes_delayed",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
