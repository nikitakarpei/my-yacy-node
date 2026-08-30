package vault_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestPutThenGetTranslatesThroughCodec(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		got, ok, err := words.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if !ok || got != "alpha" {
			t.Fatalf("Get(a) = %q, %v", got, ok)
		}

		length, err := words.Len(tx)
		if err != nil {
			return wrap(err)
		}
		if length != 1 {
			t.Fatalf("Len = %d, want 1", length)
		}

		return words.Scan(tx, vault.EveryKey(), func(_ string, value string) (bool, error) {
			if value != "alpha" {
				t.Fatalf("Scan value = %q, want alpha", value)
			}

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestEncodeErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	collection, err := v.RegisterCollection(
		vault.Name("words"),
		stringKeyCodec,
		failingEncodeCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return collection.Put(tx, "a", "alpha")
	}); err == nil {
		t.Fatal("Put with failing encode succeeded, want error")
	}
}

func TestDecodeErrorSurfaces(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	collection, err := v.RegisterCollection(
		vault.Name("words"),
		stringKeyCodec,
		failingDecodeCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return collection.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	getErr := v.View(ctx, func(tx *vault.Txn) error {
		_, _, err := collection.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}

		return nil
	})
	if getErr == nil {
		t.Fatal("Get with failing decode succeeded, want error")
	}

	scanErr := v.View(ctx, func(tx *vault.Txn) error {
		return collection.Scan(
			tx,
			vault.EveryKey(),
			func(string, string) (bool, error) { return true, nil },
		)
	})
	if scanErr == nil {
		t.Fatal("Scan with failing decode succeeded, want error")
	}
}
