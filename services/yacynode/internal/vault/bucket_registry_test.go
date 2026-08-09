package vault_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func TestRejectsDuplicateRegistration(t *testing.T) {
	v, _ := openWords(t)

	if _, err := vault.RegisterCollection(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	); err == nil {
		t.Fatal("duplicate RegisterCollection succeeded, want error")
	}
}

func TestRegisterRejectsClosedVault(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := vault.RegisterCollection(
		v,
		vault.Name("words"),
		stringKeyCodec{},
		stringValueCodec{},
	); err == nil {
		t.Fatal("RegisterCollection on closed vault succeeded, want error")
	}
}

func TestEntriesByCollectionReportsRegisteredLengths(t *testing.T) {
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	ctx := context.Background()

	words, err := vault.RegisterCollection(v, "words", stringKeyCodec{}, stringValueCodec{})
	if err != nil {
		t.Fatalf("Register words: %v", err)
	}
	if _, err := vault.RegisterCollection(
		v,
		"urls",
		stringKeyCodec{},
		stringValueCodec{},
	); err != nil {
		t.Fatalf("Register urls: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		for index := range 3 {
			if err := words.Put(tx, fmt.Sprintf("k%d", index), "v"); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	entries, err := v.EntriesByCollection(ctx)
	if err != nil {
		t.Fatalf("EntriesByCollection: %v", err)
	}

	want := map[vault.Name]int{"words": 3, "urls": 0}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for collection, count := range want {
		if entries[collection] != count {
			t.Errorf("entries[%s] = %d, want %d", collection, entries[collection], count)
		}
	}
}
