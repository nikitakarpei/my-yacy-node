// Package pebblemetrics exposes the Pebble storage engine condition, write stalls and
// write conflicts as Prometheus metrics under the namespace of the service that wires them.
package pebblemetrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	labelCompactionKind = "kind"
	labelLevel          = "level"
)

type Engine interface {
	Condition() pebblevault.EngineCondition
}

type ConditionMetrics struct {
	engine               Engine
	engineReadings       []engineReading
	levelReadings        []levelReading
	compactionsCompleted *prometheus.Desc
}

type engineReading struct {
	description *prometheus.Desc
	kind        prometheus.ValueType
	valueOf     func(pebblevault.EngineCondition) float64
}

type levelReading struct {
	description *prometheus.Desc
	kind        prometheus.ValueType
	valueOf     func(pebblevault.Level) float64
}

func NewConditionMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
	engine Engine,
) *ConditionMetrics {
	metrics := &ConditionMetrics{
		engine:         engine,
		engineReadings: engineReadings(serviceMetricNamespace),
		levelReadings:  levelReadings(serviceMetricNamespace),
		compactionsCompleted: prometheus.NewDesc(
			serviceMetricNamespace+"_pebble_compactions_completed_total",
			"Compactions the storage engine finished, by the kind of compaction.",
			[]string{labelCompactionKind},
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *ConditionMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	for _, reading := range m.engineReadings {
		descriptions <- reading.description
	}
	for _, reading := range m.levelReadings {
		descriptions <- reading.description
	}
	descriptions <- m.compactionsCompleted
}

func (m *ConditionMetrics) Collect(samples chan<- prometheus.Metric) {
	condition := m.engine.Condition()

	m.collectEngine(samples, condition)
	m.collectLevels(samples, condition)
	m.collectCompactions(samples, condition)
}

func (m *ConditionMetrics) collectEngine(
	samples chan<- prometheus.Metric,
	condition pebblevault.EngineCondition,
) {
	for _, reading := range m.engineReadings {
		samples <- prometheus.MustNewConstMetric(
			reading.description,
			reading.kind,
			reading.valueOf(condition),
		)
	}
}

func (m *ConditionMetrics) collectLevels(
	samples chan<- prometheus.Metric,
	condition pebblevault.EngineCondition,
) {
	for _, level := range condition.Levels {
		m.collectLevel(samples, level)
	}
}

func (m *ConditionMetrics) collectLevel(
	samples chan<- prometheus.Metric,
	level pebblevault.Level,
) {
	number := strconv.Itoa(level.Number)
	for _, reading := range m.levelReadings {
		samples <- prometheus.MustNewConstMetric(
			reading.description,
			reading.kind,
			reading.valueOf(level),
			number,
		)
	}
}

func (m *ConditionMetrics) collectCompactions(
	samples chan<- prometheus.Metric,
	condition pebblevault.EngineCondition,
) {
	for kind, completed := range condition.CompactionsCompleted {
		samples <- prometheus.MustNewConstMetric(
			m.compactionsCompleted,
			prometheus.CounterValue,
			float64(completed),
			string(kind),
		)
	}
}

func engineReadings(serviceMetricNamespace string) []engineReading {
	readings := compactionReadings(serviceMetricNamespace)
	readings = append(readings, memtableReadings(serviceMetricNamespace)...)
	readings = append(readings, blockCacheReadings(serviceMetricNamespace)...)
	readings = append(readings, fileCacheReadings(serviceMetricNamespace)...)
	readings = append(readings, bloomFilterReadings(serviceMetricNamespace)...)
	readings = append(readings, diskFootprintReadings(serviceMetricNamespace)...)
	readings = append(readings, amountScanReadings(serviceMetricNamespace)...)

	return readings
}

func compactionReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_compaction_debt_bytes",
			"Bytes the storage engine estimates it must still compact to settle its shape.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.CompactionDebtBytes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_compactions_in_progress",
			"Compactions the storage engine is running now.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.CompactionsInProgress) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_compaction_concurrency_limit",
			"Compactions the storage engine may run at the same time.",
			func(c pebblevault.EngineCondition) float64 {
				return float64(c.CompactionConcurrencyLimit)
			},
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_compactions_failed_total",
			"Compactions that ended in an error.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.CompactionsFailed) },
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_compaction_seconds_total",
			"Time the storage engine spent in compaction, in seconds.",
			func(c pebblevault.EngineCondition) float64 { return c.CompactionSeconds },
		),
	}
}

func memtableReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineCounterReading(
			serviceMetricNamespace+"_pebble_flushes_total",
			"Write buffers the storage engine wrote out to level zero.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.Flushes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_memtable_held_bytes",
			"Bytes the write buffers hold in memory.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.MemtableHeldBytes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_memtables",
			"Write buffers the storage engine holds, including those waiting to be written out.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.Memtables) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_memtable_size_limit_bytes",
			"Bytes one write buffer holds before the storage engine writes it out.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.MemtableSizeLimitBytes) },
		),
	}
}

func blockCacheReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineCounterReading(
			serviceMetricNamespace+"_pebble_block_cache_hits_total",
			"Data block reads the block cache answered from memory.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BlockCacheHits) },
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_block_cache_misses_total",
			"Data block reads the block cache could not answer, so a table file was read.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BlockCacheMisses) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_block_cache_held_bytes",
			"Bytes of data blocks the block cache holds.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BlockCacheHeldBytes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_block_cache_limit_bytes",
			"Bytes the block cache may hold.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BlockCacheLimitBytes) },
		),
	}
}

func fileCacheReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineCounterReading(
			serviceMetricNamespace+"_pebble_file_cache_hits_total",
			"Table file reads the file cache answered from an already open file.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.FileCacheHits) },
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_file_cache_misses_total",
			"Table file reads the file cache could not answer, so a file was opened.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.FileCacheMisses) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_file_cache_tables",
			"Table files the file cache holds open.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.FileCacheTables) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_file_cache_limit",
			"Table files the file cache may hold open.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.FileCacheLimit) },
		),
	}
}

func bloomFilterReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineCounterReading(
			serviceMetricNamespace+"_pebble_bloom_filter_hits_total",
			"Data block reads a bloom filter made unnecessary.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BloomFilterHits) },
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_bloom_filter_misses_total",
			"Data block reads a bloom filter could not make unnecessary.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.BloomFilterMisses) },
		),
	}
}

func diskFootprintReadings(serviceMetricNamespace string) []engineReading {
	return append(reclaimableReadings(serviceMetricNamespace), []engineReading{
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_write_ahead_log_bytes",
			"Bytes the write-ahead log files occupy on disk.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.WriteAheadLogBytes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_write_ahead_log_files",
			"Write-ahead log files the storage engine holds.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.WriteAheadLogFiles) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_obsolete_table_bytes",
			"Bytes in table files no longer referenced and waiting to be deleted.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.ObsoleteTableBytes) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_zombie_table_bytes",
			"Bytes in table files no longer referenced but still held open by a reader.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.ZombieTableBytes) },
		),
	}...)
}

func reclaimableReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_tombstone_keys",
			"Deletion markers the storage engine holds until a compaction removes them.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.TombstoneKeys) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_point_deletion_reclaimable_bytes",
			"Bytes a compaction of all single-key deletions would free.",
			func(c pebblevault.EngineCondition) float64 {
				return float64(c.PointDeletionReclaimableBytes)
			},
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_range_deletion_reclaimable_bytes",
			"Bytes a compaction of all key-range deletions would free.",
			func(c pebblevault.EngineCondition) float64 {
				return float64(c.RangeDeletionReclaimableBytes)
			},
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_open_snapshots",
			"Read snapshots open now. Each one holds back the bytes it can still see.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.OpenSnapshots) },
		),
		engineGaugeReading(
			serviceMetricNamespace+"_pebble_snapshot_pinned_bytes",
			"Bytes a compaction kept because an open snapshot can still see them.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.SnapshotPinnedBytes) },
		),
	}
}

func amountScanReadings(serviceMetricNamespace string) []engineReading {
	return []engineReading{
		engineCounterReading(
			serviceMetricNamespace+"_pebble_amounts_scanned_total",
			"Stored amounts that scans stepped over, at zero or above.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.AmountsScanned) },
		),
		engineCounterReading(
			serviceMetricNamespace+"_pebble_amounts_at_zero_scanned_total",
			"Stored amounts at zero that scans stepped over and skipped.",
			func(c pebblevault.EngineCondition) float64 { return float64(c.AmountsAtZeroScanned) },
		),
	}
}

func engineGaugeReading(
	name, help string,
	valueOf func(pebblevault.EngineCondition) float64,
) engineReading {
	return engineReading{
		description: prometheus.NewDesc(name, help, nil, nil),
		kind:        prometheus.GaugeValue,
		valueOf:     valueOf,
	}
}

func engineCounterReading(
	name, help string,
	valueOf func(pebblevault.EngineCondition) float64,
) engineReading {
	return engineReading{
		description: prometheus.NewDesc(name, help, nil, nil),
		kind:        prometheus.CounterValue,
		valueOf:     valueOf,
	}
}

func levelReadings(serviceMetricNamespace string) []levelReading {
	return []levelReading{
		levelGaugeReading(
			serviceMetricNamespace+"_pebble_level_sublevels",
			"Sublevels at this level. Their sum over all levels is the read amplification.",
			func(l pebblevault.Level) float64 { return float64(l.Sublevels) },
		),
		levelGaugeReading(
			serviceMetricNamespace+"_pebble_level_tables",
			"Table files at this level.",
			func(l pebblevault.Level) float64 { return float64(l.Tables) },
		),
		levelGaugeReading(
			serviceMetricNamespace+"_pebble_level_table_bytes",
			"Bytes in the table files at this level.",
			func(l pebblevault.Level) float64 { return float64(l.TableBytes) },
		),
		levelCounterReading(
			serviceMetricNamespace+"_pebble_level_incoming_bytes_total",
			"Bytes compactions brought into this level from other levels.",
			func(l pebblevault.Level) float64 { return float64(l.IncomingBytes) },
		),
		levelCounterReading(
			serviceMetricNamespace+"_pebble_level_compacted_bytes_total",
			"Bytes compactions wrote to table files at this level.",
			func(l pebblevault.Level) float64 { return float64(l.CompactedBytes) },
		),
		levelCounterReading(
			serviceMetricNamespace+"_pebble_level_flushed_bytes_total",
			"Bytes write buffers wrote to table files at this level.",
			func(l pebblevault.Level) float64 { return float64(l.FlushedBytes) },
		),
	}
}

func levelGaugeReading(name, help string, valueOf func(pebblevault.Level) float64) levelReading {
	return levelReading{
		description: prometheus.NewDesc(name, help, []string{labelLevel}, nil),
		kind:        prometheus.GaugeValue,
		valueOf:     valueOf,
	}
}

func levelCounterReading(name, help string, valueOf func(pebblevault.Level) float64) levelReading {
	return levelReading{
		description: prometheus.NewDesc(name, help, []string{labelLevel}, nil),
		kind:        prometheus.CounterValue,
		valueOf:     valueOf,
	}
}
