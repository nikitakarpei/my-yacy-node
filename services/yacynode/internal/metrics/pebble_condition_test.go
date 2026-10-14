package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type stubPebbleEngine struct {
	condition pebblevault.EngineCondition
}

func (s stubPebbleEngine) Condition() pebblevault.EngineCondition {
	return s.condition
}

func TestPebbleConditionReportsTheLimitsBesideTheReadings(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewPebbleConditionMetrics(registry, stubPebbleEngine{
		condition: pebblevault.EngineCondition{
			BlockCacheHeldBytes:  512,
			BlockCacheLimitBytes: 2048,
		},
	})

	expected := `
# HELP yacynode_pebble_block_cache_held_bytes Bytes of data blocks the block cache holds.
# TYPE yacynode_pebble_block_cache_held_bytes gauge
yacynode_pebble_block_cache_held_bytes 512
# HELP yacynode_pebble_block_cache_limit_bytes Bytes the block cache may hold.
# TYPE yacynode_pebble_block_cache_limit_bytes gauge
yacynode_pebble_block_cache_limit_bytes 2048
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_block_cache_held_bytes",
		"yacynode_pebble_block_cache_limit_bytes",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestPebbleConditionReportsCompactionsByKind(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewPebbleConditionMetrics(registry, stubPebbleEngine{
		condition: pebblevault.EngineCondition{
			CompactionsCompleted: map[pebblevault.CompactionKind]int64{"default": 7, "move": 2},
		},
	})

	expected := `
# HELP yacynode_pebble_compactions_completed_total Compactions the storage engine finished, by the kind of compaction.
# TYPE yacynode_pebble_compactions_completed_total counter
yacynode_pebble_compactions_completed_total{kind="default"} 7
yacynode_pebble_compactions_completed_total{kind="move"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_compactions_completed_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestPebbleConditionReportsEachLevelSeparately(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewPebbleConditionMetrics(registry, stubPebbleEngine{
		condition: pebblevault.EngineCondition{
			Levels: []pebblevault.Level{
				{Number: 0, Sublevels: 3, Tables: 12},
				{Number: 1, Sublevels: 1, Tables: 4},
			},
		},
	})

	expected := `
# HELP yacynode_pebble_level_tables Table files at this level.
# TYPE yacynode_pebble_level_tables gauge
yacynode_pebble_level_tables{level="0"} 12
yacynode_pebble_level_tables{level="1"} 4
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_level_tables",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestPebbleConditionIsReadAgainOnEveryScrape(t *testing.T) {
	registry := prometheus.NewRegistry()
	condition := &countingPebbleEngine{}
	metrics.NewPebbleConditionMetrics(registry, condition)

	for range 3 {
		if _, err := registry.Gather(); err != nil {
			t.Fatalf("Gather: %v", err)
		}
	}

	if condition.reads != 3 {
		t.Errorf("engine read %d times, want one read per scrape", condition.reads)
	}
}

type countingPebbleEngine struct {
	reads int
}

func (c *countingPebbleEngine) Condition() pebblevault.EngineCondition {
	c.reads++

	return pebblevault.EngineCondition{}
}

func TestPebbleConditionReportsTheAmountsScansSteppedOver(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewPebbleConditionMetrics(registry, stubPebbleEngine{
		condition: pebblevault.EngineCondition{AmountsScanned: 7, AmountsAtZeroScanned: 3},
	})

	expected := `
# HELP yacynode_pebble_amounts_at_zero_scanned_total Stored amounts at zero that scans stepped over and skipped.
# TYPE yacynode_pebble_amounts_at_zero_scanned_total counter
yacynode_pebble_amounts_at_zero_scanned_total 3
# HELP yacynode_pebble_amounts_scanned_total Stored amounts that scans stepped over, at zero or above.
# TYPE yacynode_pebble_amounts_scanned_total counter
yacynode_pebble_amounts_scanned_total 7
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_pebble_amounts_scanned_total",
		"yacynode_pebble_amounts_at_zero_scanned_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
