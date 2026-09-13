package boltvault_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault"
)

var stringKeyParts = vault.SingleKey(vault.TextKeyPart)

var stringKeyLayout = stringKeyParts.KeyLayout()

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }

func openEngine(t *testing.T) vault.Engine {
	t.Helper()

	engine, err := boltvault.OpenEngine(
		filepath.Join(t.TempDir(), "node.db"),
		0,
		boltvault.WriteBatch{},
	)
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

func TestConformance(t *testing.T) {
	dir := t.TempDir()
	var seq atomic.Int64

	vaultenginetest.RunConformance(t, func(quotaBytes int64) (vault.Engine, error) {
		path := filepath.Join(dir, fmt.Sprintf("node-%d.db", seq.Add(1)))

		return boltvault.OpenEngine(path, quotaBytes, boltvault.WriteBatch{})
	})
}

func TestDurabilityAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "node.db")

	first, err := boltvault.Open(path, 0, boltvault.WriteBatch{}, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	words, err := first.RegisterCollection(
		vault.Name("words"),
		stringKeyLayout,
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

	reopened, err := boltvault.Open(path, 0, boltvault.WriteBatch{}, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	words, err = reopened.RegisterCollection(
		vault.Name("words"),
		stringKeyLayout,
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

func TestOpenRefusesAStoragePathUnderAFile(t *testing.T) {
	occupied := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(occupied, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := boltvault.Open(
		filepath.Join(occupied, "node.db"),
		0,
		boltvault.WriteBatch{},
		nil,
	); err == nil {
		t.Fatal("Open under a file succeeded, want error")
	}
}

func TestOpenRefusesAStoragePathThatIsADirectory(t *testing.T) {
	if _, err := boltvault.OpenEngine(t.TempDir(), 0, boltvault.WriteBatch{}); err == nil {
		t.Fatal("OpenEngine on a directory succeeded, want error")
	}
}

func TestProvisionRefusesACollectionWithNoName(t *testing.T) {
	if err := openEngine(t).Provision(""); err == nil {
		t.Fatal("Provision of an unnamed collection succeeded, want error")
	}
}

func TestProvisionRefusesTheReservedLengthCollection(t *testing.T) {
	if err := openEngine(t).Provision("__lengths__"); err == nil {
		t.Fatal("Provision of the reserved collection succeeded, want error")
	}
}

func TestWritesAreNotGatedByQuota(t *testing.T) {
	ctx := context.Background()
	store, err := boltvault.Open(
		filepath.Join(t.TempDir(), "node.db"),
		1,
		boltvault.WriteBatch{},
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
	words, err := store.RegisterCollection(
		vault.Name("words"),
		stringKeyLayout,
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
