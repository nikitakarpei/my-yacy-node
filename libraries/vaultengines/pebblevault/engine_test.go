package pebblevault_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const testCacheBytes = 8 << 20

var stringKeyLayout = vault.SingleKey(vault.TextKeyPart).KeyLayout()

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }

func openVault(t *testing.T, quotaBytes int64) *vault.Vault {
	t.Helper()

	store, err := pebblevault.Open(
		filepath.Join(t.TempDir(), "node"),
		quotaBytes,
		testCacheBytes,
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

	return store
}

func registerWords(t *testing.T, store *vault.Vault) *vault.Collection[string, string] {
	t.Helper()

	words, err := store.RegisterCollection(
		vault.Name("words"),
		stringKeyLayout,
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	return words
}

func TestConformance(t *testing.T) {
	vaultenginetest.RunConformance(t, func(quotaBytes int64) (vault.Engine, error) {
		opened, err := pebblevault.OpenEngine(t.TempDir()+"/node", quotaBytes, testCacheBytes)
		if err != nil {
			return nil, err
		}

		return vaultenginetest.EngineRepeatingWrites(opened), nil
	})
}

func TestDurabilityAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node")

	first, err := pebblevault.Open(path, 0, testCacheBytes, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	words := registerWords(t, first)
	if err := first.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := pebblevault.Open(path, 0, testCacheBytes, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	words = registerWords(t, reopened)

	if err := reopened.View(ctx, func(tx *vault.Txn) error {
		got, ok, err := words.Get(tx, "a")
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}
		if !ok || got != "alpha" {
			t.Fatalf("after reopen Get(a) = %q, %v", got, ok)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestOpenRefusesAStoragePathUnderAFile(t *testing.T) {
	occupied := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(occupied, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := pebblevault.Open(
		filepath.Join(occupied, "node"),
		0,
		testCacheBytes,
		nil,
	); err == nil {
		t.Fatal("Open under a file succeeded, want error")
	}
}

func TestOpenRefusesAStoragePathAnotherProcessHolds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node")
	openVaultAt(t, path)

	if _, err := pebblevault.OpenEngine(path, 0, testCacheBytes); err == nil {
		t.Fatal("OpenEngine on a held directory succeeded, want error")
	}
}

func openVaultAt(t *testing.T, path string) {
	t.Helper()

	holder, err := pebblevault.OpenEngine(path, 0, testCacheBytes)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	t.Cleanup(func() {
		if err := holder.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
}

func TestWritesAreNotGatedByQuota(t *testing.T) {
	store := openVault(t, 1)
	words := registerWords(t, store)

	if err := store.Update(context.Background(), func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update over quota = %v, want nil (the engine does not gate writes)", err)
	}
}
