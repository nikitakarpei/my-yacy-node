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

func TestIndexEntriesCountTowardUsedBytes(t *testing.T) {
	plain := openVault(t, 0)
	plainWords := registerWords(t, plain)
	indexed := openVault(t, 0)
	indexedWords := registerIndexedWords(t, indexed)

	plainGrowth := usedBytesGrowthOf(t, plain, func(tx *vault.Txn) error {
		_, err := plainWords.Put(tx, "a", "alpha")

		return err
	})
	indexedGrowth := usedBytesGrowthOf(t, indexed, func(tx *vault.Txn) error {
		_, err := indexedWords.Put(tx, "a", "alpha")

		return err
	})

	if indexedGrowth <= plainGrowth {
		t.Fatalf(
			"UsedBytes rose by %d with an index, want more than %d",
			indexedGrowth,
			plainGrowth,
		)
	}
}

func TestDeletedRecordsGiveBackTheBytesOfTheirIndexEntries(t *testing.T) {
	store := openVault(t, 0)
	words := registerIndexedWords(t, store)

	growth := usedBytesGrowthOf(t, store, func(tx *vault.Txn) error {
		for _, value := range []string{"alpha", "alpha-and-then-some"} {
			if _, err := words.Put(tx, "a", value); err != nil {
				return err
			}
		}
		if _, err := words.Put(tx, "b", "beta"); err != nil {
			return err
		}
		if _, err := words.Delete(tx, "a"); err != nil {
			return err
		}
		_, err := words.Delete(tx, "b")

		return err
	})

	if growth != 0 {
		t.Fatalf("UsedBytes changed by %d after every record left, want 0", growth)
	}
}

func registerIndexedWords(t *testing.T, store *vault.Vault) *vault.Collection[string, string] {
	t.Helper()

	byValue, err := store.RegisterValueIndex(
		vault.Name("words_by_value"),
		"1",
		stringKeyLayout,
		func(entry string) string { return entry },
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	words, err := store.RegisterProjectedCollection(
		context.Background(),
		vault.Name("words"),
		stringKeyLayout,
		stringValueCodec{},
		vault.ProjectedAs(func(word, value string) string { return value + " " + word }, byValue),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return words
}

func usedBytesGrowthOf(t *testing.T, store *vault.Vault, change func(tx *vault.Txn) error) int64 {
	t.Helper()

	ctx := context.Background()
	before, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if err := store.Update(ctx, change); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	return after - before
}
