package pebblevault_test

import (
	"context"
	"crypto/sha256"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const smallMemtableBytes = 32 << 10

func TestLevelZeroCollectsManySublevelsBeforeACompaction(t *testing.T) {
	engine := openEngine(t, pebblevault.MachineLimits{MemtableBytes: smallMemtableBytes})

	peakSublevels := 0
	for batch := range 24 {
		storeScatteredWords(t, engine, batch, 128)
		peakSublevels = max(peakSublevels, engine.Condition().Levels[0].Sublevels)
	}

	if peakSublevels < 6 {
		t.Errorf("level 0 peaked at %d sublevels, want at least 6", peakSublevels)
	}
}

func TestDeletionsLeaveNoTombstoneDensityCompactions(t *testing.T) {
	engine := openEngine(t, pebblevault.MachineLimits{MemtableBytes: 64 << 10})

	for batch := range 8 {
		storeWords(t, engine, batch, 256)
	}
	for batch := range 8 {
		deleteWords(t, engine, batch)
	}
	for batch := range 4 {
		storeWords(t, engine, 100+batch, 256)
	}
	settled := conditionMeeting(t, engine, func(c pebblevault.EngineCondition) bool {
		return c.CompactionsInProgress == 0
	})

	if compactions := settled.CompactionsCompleted["tombstone_density"]; compactions != 0 {
		t.Errorf("tombstone density compactions = %d, want 0", compactions)
	}
}

func storeScatteredWords(t *testing.T, engine *pebblevault.Engine, batch, words int) {
	t.Helper()

	value := make([]byte, 64)
	if err := engine.Update(context.Background(), func(tx vault.EngineTxn) error {
		bucket := tx.Records(vault.Name("words"))
		for word := range words {
			key := sha256.Sum256([]byte(strconv.Itoa(batch) + "-" + strconv.Itoa(word)))
			copy(value, key[:])
			if _, err := bucket.Put(key[:12], value); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func deleteWords(t *testing.T, engine *pebblevault.Engine, round int) {
	t.Helper()

	if err := engine.Update(context.Background(), func(tx vault.EngineTxn) error {
		bucket := tx.Records(vault.Name("words"))
		for word := range 256 {
			if _, err := bucket.Delete(
				[]byte(strconv.Itoa(round) + "-" + strconv.Itoa(word)),
			); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}
