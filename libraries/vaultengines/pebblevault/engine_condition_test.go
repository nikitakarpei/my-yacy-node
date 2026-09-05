package pebblevault_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const levelsInTheTree = 7

func TestConditionReportsTheLimitsTheEngineImposes(t *testing.T) {
	condition := openEngine(t, pebblevault.MachineLimits{
		BlockCacheBytes:       8 << 20,
		MemtableBytes:         1 << 20,
		CompactionConcurrency: 3,
		OpenFileLimit:         64,
	}).Condition()

	if condition.BlockCacheLimitBytes != 8<<20 {
		t.Errorf("BlockCacheLimitBytes = %d, want %d", condition.BlockCacheLimitBytes, 8<<20)
	}
	if condition.MemtableSizeLimitBytes != 1<<20 {
		t.Errorf("MemtableSizeLimitBytes = %d, want %d", condition.MemtableSizeLimitBytes, 1<<20)
	}
	if condition.CompactionConcurrencyLimit != 3 {
		t.Errorf("CompactionConcurrencyLimit = %d, want 3", condition.CompactionConcurrencyLimit)
	}
	if condition.FileCacheLimit != 64 {
		t.Errorf("FileCacheLimit = %d, want 64", condition.FileCacheLimit)
	}
}

func TestConditionReportsTheLimitsTheEngineChoosesForItself(t *testing.T) {
	condition := openEngine(t, pebblevault.MachineLimits{}).Condition()

	if condition.BlockCacheLimitBytes <= 0 {
		t.Errorf(
			"BlockCacheLimitBytes = %d, want a positive default",
			condition.BlockCacheLimitBytes,
		)
	}
	if condition.MemtableSizeLimitBytes <= 0 {
		t.Errorf(
			"MemtableSizeLimitBytes = %d, want a positive default",
			condition.MemtableSizeLimitBytes,
		)
	}
	if condition.CompactionConcurrencyLimit <= 0 {
		t.Errorf(
			"CompactionConcurrencyLimit = %d, want a positive default",
			condition.CompactionConcurrencyLimit,
		)
	}
	if condition.FileCacheLimit <= 0 {
		t.Errorf("FileCacheLimit = %d, want a positive default", condition.FileCacheLimit)
	}
}

func TestConditionReportsEveryLevelOfTheTree(t *testing.T) {
	levels := openEngine(t, testLimits).Condition().Levels

	if len(levels) != levelsInTheTree {
		t.Fatalf("levels = %d, want %d", len(levels), levelsInTheTree)
	}
	for number, level := range levels {
		if level.Number != number {
			t.Errorf("level at %d reports number %d", number, level.Number)
		}
	}
}

func TestConditionReportsTheTablesTheEngineWritesOut(t *testing.T) {
	engine := openEngine(t, pebblevault.MachineLimits{MemtableBytes: 256 << 10})

	before := engine.Condition()
	for batch := range 4 {
		storeWords(t, engine, batch, 1024)
	}
	settled := conditionMeeting(t, engine, func(c pebblevault.EngineCondition) bool {
		return c.Flushes > before.Flushes
	})

	if settled.Levels[0].Tables <= before.Levels[0].Tables {
		t.Errorf(
			"level 0 tables went %d -> %d, want a rise",
			before.Levels[0].Tables,
			settled.Levels[0].Tables,
		)
	}
	if settled.Levels[0].TableBytes <= before.Levels[0].TableBytes {
		t.Errorf(
			"level 0 table bytes went %d -> %d, want a rise",
			before.Levels[0].TableBytes,
			settled.Levels[0].TableBytes,
		)
	}
}

func conditionMeeting(
	t *testing.T,
	engine *pebblevault.Engine,
	met func(pebblevault.EngineCondition) bool,
) pebblevault.EngineCondition {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for {
		condition := engine.Condition()
		if met(condition) {
			return condition
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"engine never settled: flushes = %d, level zero tables = %d",
				condition.Flushes,
				condition.Levels[0].Tables,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestConditionReportsTheSnapshotAReadHoldsOpen(t *testing.T) {
	engine := openEngine(t, testLimits)

	var duringRead int64
	if err := engine.View(context.Background(), func(vault.EngineTxn) error {
		duringRead = engine.Condition().OpenSnapshots

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if duringRead < 1 {
		t.Errorf("OpenSnapshots during a read = %d, want at least 1", duringRead)
	}
	if after := engine.Condition().OpenSnapshots; after != 0 {
		t.Errorf("OpenSnapshots after a read = %d, want 0", after)
	}
}

func openEngine(t *testing.T, limits pebblevault.MachineLimits) *pebblevault.Engine {
	t.Helper()

	engine, err := pebblevault.OpenEngine(filepath.Join(t.TempDir(), "node"), 0, limits, nil)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return engine
}

func storeWords(t *testing.T, engine *pebblevault.Engine, batch, words int) {
	t.Helper()

	value := bytes.Repeat([]byte("a"), 1024)
	if err := engine.Update(context.Background(), func(tx vault.EngineTxn) error {
		bucket := tx.Bucket(vault.Name("words"))
		for word := range words {
			key := strconv.Itoa(batch) + "-" + strconv.Itoa(word)
			if err := bucket.Put([]byte(key), value); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}
