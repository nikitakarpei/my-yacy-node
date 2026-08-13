package bolt_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/bolt"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaulttest"
)

var stringKeyLayout = vaultkey.Single(vaultkey.Text)

type stringKeyCodec struct{}

func (stringKeyCodec) Encode(key string) vaultkey.Key { return stringKeyLayout.Key(key) }

func (stringKeyCodec) Decode(storedKey []byte) (string, error) {
	decoded, err := stringKeyLayout.Parts(storedKey)
	if err != nil {
		return "", fmt.Errorf("word key: %w", err)
	}

	return decoded, nil
}

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }

func TestConformance(t *testing.T) {
	dir := t.TempDir()
	var seq atomic.Int64

	vaulttest.RunConformance(t, func(quotaBytes int64) (vault.Engine, error) {
		path := filepath.Join(dir, fmt.Sprintf("node-%d.db", seq.Add(1)))

		return bolt.OpenEngine(path, quotaBytes)
	})
}

func TestDurabilityAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")

	first, err := bolt.Open(path, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	words, err := vault.RegisterCollection(
		first,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := first.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := bolt.Open(path, 0, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	words, err = vault.RegisterCollection(
		reopened,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("re-register: %v", err)
	}

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

func TestWritesAreNotGatedByQuota(t *testing.T) {
	ctx := context.Background()
	store, err := bolt.Open(filepath.Join(t.TempDir(), "node.db"), 1, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	words, err := vault.RegisterCollection(
		store,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := store.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update over quota = %v, want nil (kernel does not gate writes)", err)
	}
}
