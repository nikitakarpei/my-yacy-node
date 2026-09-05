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
	store, err := pebblevault.Open(
		filepath.Join(t.TempDir(), "node"),
		4096,
		8<<20,
		nil,
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
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
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	short, err := store.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if err := store.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha-and-then-some")
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

func TestUsedBytesStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := openVault(t, 0).UsedBytes(ctx); err == nil {
		t.Fatal("UsedBytes on a cancelled context succeeded, want error")
	}
}
