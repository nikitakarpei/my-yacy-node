package pebblevault

import "github.com/cockroachdb/pebble/v2"

type CompactionKind string

const (
	kindDefault          CompactionKind = "default"
	kindDeleteOnly       CompactionKind = "delete_only"
	kindElisionOnly      CompactionKind = "elision_only"
	kindCopy             CompactionKind = "copy"
	kindMove             CompactionKind = "move"
	kindRead             CompactionKind = "read"
	kindTombstoneDensity CompactionKind = "tombstone_density"
	kindRewrite          CompactionKind = "rewrite"
	kindMultiLevel       CompactionKind = "multi_level"
)

type EngineCondition struct {
	CompactionDebtBytes           int64
	CompactionsInProgress         int64
	CompactionConcurrencyLimit    int
	CompactionsCompleted          map[CompactionKind]int64
	CompactionsFailed             int64
	CompactionSeconds             float64
	Flushes                       int64
	MemtableHeldBytes             int64
	Memtables                     int64
	MemtableSizeLimitBytes        int64
	Levels                        []Level
	BlockCacheHits                int64
	BlockCacheMisses              int64
	BlockCacheHeldBytes           int64
	BlockCacheLimitBytes          int64
	FileCacheHits                 int64
	FileCacheMisses               int64
	FileCacheTables               int64
	FileCacheLimit                int
	BloomFilterHits               int64
	BloomFilterMisses             int64
	DiskOccupiedBytes             int64
	WriteAheadLogBytes            int64
	WriteAheadLogFiles            int64
	ObsoleteTableBytes            int64
	ZombieTableBytes              int64
	TombstoneKeys                 int64
	PointDeletionReclaimableBytes int64
	RangeDeletionReclaimableBytes int64
	OpenSnapshots                 int64
	SnapshotPinnedBytes           int64
}

type Level struct {
	Number         int
	Sublevels      int
	Tables         int64
	TableBytes     int64
	IncomingBytes  int64
	CompactedBytes int64
	FlushedBytes   int64
}

func engineConditionOf(reported *pebble.Metrics, limits MachineLimits) EngineCondition {
	return EngineCondition{
		CompactionDebtBytes:           signed(reported.Compact.EstimatedDebt),
		CompactionsInProgress:         reported.Compact.NumInProgress,
		CompactionConcurrencyLimit:    limits.CompactionConcurrency,
		CompactionsCompleted:          compactionsCompletedIn(reported),
		CompactionsFailed:             reported.Compact.FailedCount,
		CompactionSeconds:             reported.Compact.Duration.Seconds(),
		Flushes:                       reported.Flush.Count,
		MemtableHeldBytes:             signed(reported.MemTable.Size),
		Memtables:                     reported.MemTable.Count,
		MemtableSizeLimitBytes:        limits.MemtableBytes,
		Levels:                        levelsIn(reported),
		BlockCacheHits:                reported.BlockCache.Hits,
		BlockCacheMisses:              reported.BlockCache.Misses,
		BlockCacheHeldBytes:           reported.BlockCache.Size,
		BlockCacheLimitBytes:          limits.BlockCacheBytes,
		FileCacheHits:                 reported.FileCache.Hits,
		FileCacheMisses:               reported.FileCache.Misses,
		FileCacheTables:               reported.FileCache.TableCount,
		FileCacheLimit:                limits.OpenFileLimit,
		BloomFilterHits:               reported.Filter.Hits,
		BloomFilterMisses:             reported.Filter.Misses,
		DiskOccupiedBytes:             signed(reported.DiskSpaceUsage()),
		WriteAheadLogBytes:            signed(reported.WAL.PhysicalSize),
		WriteAheadLogFiles:            reported.WAL.Files,
		ObsoleteTableBytes:            signed(reported.Table.ObsoleteSize),
		ZombieTableBytes:              signed(reported.Table.ZombieSize),
		TombstoneKeys:                 signed(reported.Keys.TombstoneCount),
		PointDeletionReclaimableBytes: signed(reported.Table.Garbage.PointDeletionsBytesEstimate),
		RangeDeletionReclaimableBytes: signed(reported.Table.Garbage.RangeDeletionsBytesEstimate),
		OpenSnapshots:                 int64(reported.Snapshots.Count),
		SnapshotPinnedBytes:           signed(reported.Snapshots.PinnedSize),
	}
}

func compactionsCompletedIn(reported *pebble.Metrics) map[CompactionKind]int64 {
	return map[CompactionKind]int64{
		kindDefault:          reported.Compact.DefaultCount,
		kindDeleteOnly:       reported.Compact.DeleteOnlyCount,
		kindElisionOnly:      reported.Compact.ElisionOnlyCount,
		kindCopy:             reported.Compact.CopyCount,
		kindMove:             reported.Compact.MoveCount,
		kindRead:             reported.Compact.ReadCount,
		kindTombstoneDensity: reported.Compact.TombstoneDensityCount,
		kindRewrite:          reported.Compact.RewriteCount,
		kindMultiLevel:       reported.Compact.MultiLevelCount,
	}
}

func levelsIn(reported *pebble.Metrics) []Level {
	levels := make([]Level, 0, len(reported.Levels))
	for number := range reported.Levels {
		levels = append(levels, levelOf(number, &reported.Levels[number]))
	}

	return levels
}

func levelOf(number int, reported *pebble.LevelMetrics) Level {
	return Level{
		Number:         number,
		Sublevels:      int(reported.Sublevels),
		Tables:         reported.TablesCount,
		TableBytes:     reported.TablesSize,
		IncomingBytes:  signed(reported.TableBytesIn),
		CompactedBytes: signed(reported.TableBytesCompacted),
		FlushedBytes:   signed(reported.TableBytesFlushed),
	}
}

//nolint:gosec // Pebble reports magnitudes that never reach the signed range.
func signed(magnitude uint64) int64 {
	return int64(magnitude)
}
