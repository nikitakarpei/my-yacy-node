package vault_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func openMembers(t *testing.T) (*vault.Vault, *vault.Set[string]) {
	t.Helper()

	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	members, err := vault.RegisterSet(v, vault.Name("members"), stringKeyCodec{})
	if err != nil {
		t.Fatalf("RegisterSet: %v", err)
	}

	return v, members
}

func TestAddedKeysAreScannedAndCounted(t *testing.T) {
	ctx := context.Background()
	v, members := openMembers(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return members.Add(tx, "a")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		length, err := members.Len(tx)
		if err != nil {
			return wrap(err)
		}
		if length != 1 {
			t.Fatalf("Len = %d, want 1", length)
		}

		var scanned []string

		if err := members.Scan(tx, vaultkey.EveryKey(), func(key string) (bool, error) {
			scanned = append(scanned, key)

			return true, nil
		}); err != nil {
			return wrap(err)
		}
		if len(scanned) != 1 || scanned[0] != "a" {
			t.Fatalf("Scan keys = %v, want [a]", scanned)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestRemovedKeyLeavesSetEmpty(t *testing.T) {
	ctx := context.Background()
	v, members := openMembers(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if err := members.Add(tx, "a"); err != nil {
			return wrap(err)
		}

		removed, err := members.Remove(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if !removed {
			t.Fatal("Remove(a) = false, want true")
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		length, err := members.Len(tx)
		if err != nil {
			return wrap(err)
		}
		if length != 0 {
			t.Fatalf("Len = %d, want 0", length)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestRegisterSetRejectsDuplicateBucket(t *testing.T) {
	v, _ := openMembers(t)

	if _, err := vault.RegisterSet(v, vault.Name("members"), stringKeyCodec{}); err == nil {
		t.Fatal("duplicate RegisterSet succeeded, want error")
	}
}
