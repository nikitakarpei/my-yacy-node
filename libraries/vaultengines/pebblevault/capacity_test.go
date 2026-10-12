package pebblevault_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

func TestQuotaAndUsedBytes(t *testing.T) {
	ctx := context.Background()
	store, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "node"),
		4096,
		testLimits,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if store.QuotaBytes() != 4096 {
		t.Fatalf("QuotaBytes = %d, want 4096", store.QuotaBytes())
	}

	used, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if used < 0 {
		t.Fatalf("UsedBytes = %d, want non-negative", used)
	}
}

func TestUsedBytesFollowsTheValueAnOverwriteLeaves(t *testing.T) {
	ctx := context.Background()
	store := openVault(t, 0)
	words := registerWords(t, store)

	if err := store.Update(ctx, func(tx *vault.Txn) error {
		_, err := words.Put(tx, "a", "alpha")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	short, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if err := store.Update(ctx, func(tx *vault.Txn) error {
		_, err := words.Put(tx, "a", "alpha-and-then-some")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	long, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	grown := int64(len("alpha-and-then-some") - len("alpha"))
	if long-short != grown {
		t.Fatalf("UsedBytes rose by %d, want %d", long-short, grown)
	}
}

func TestUsedBytesCountsOnlyWhatATransactionLeaves(t *testing.T) {
	ctx := context.Background()
	rewritten := openVault(t, 0)
	rewrittenWords := registerWords(t, rewritten)
	written := openVault(t, 0)
	writtenWords := registerWords(t, written)

	if err := rewritten.Update(ctx, func(tx *vault.Txn) error {
		return rewriteWords(tx, rewrittenWords)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := written.Update(ctx, func(tx *vault.Txn) error {
		_, err := writtenWords.Put(tx, "a", "beta")

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	rewrittenBytes, err := rewritten.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	writtenBytes, err := written.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if rewrittenBytes != writtenBytes {
		t.Fatalf(
			"UsedBytes = %d, want %d, as for the value the transaction leaves",
			rewrittenBytes,
			writtenBytes,
		)
	}
}

func rewriteWords(tx *vault.Txn, words *vault.Collection[string, string]) error {
	for _, value := range []string{"alpha", "alpha-and-then-some", "beta"} {
		if _, _, err := words.Get(tx, "a"); err != nil {
			return err
		}
		if _, err := words.Put(tx, "a", value); err != nil {
			return err
		}
	}
	for range 2 {
		if _, err := words.Put(tx, "b", "gamma"); err != nil {
			return err
		}
		if _, err := words.Delete(tx, "b"); err != nil {
			return err
		}
	}

	return nil
}

func TestUsedBytesStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := openVault(t, 0).UsedBytes(ctx); err == nil {
		t.Fatal("UsedBytes on a cancelled context succeeded, want error")
	}
}
