package pebblevault_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

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
