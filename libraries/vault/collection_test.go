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
		_, _, storeErr := words.Put(tx, "a", "alpha")

		return storeErr
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
		stringKeyLayout,
		failingEncodeCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		_, _, storeErr := collection.Put(tx, "a", "alpha")

		return storeErr
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
		stringKeyLayout,
		failingDecodeCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		_, _, storeErr := collection.Put(tx, "a", "alpha")

		return storeErr
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

func TestPutReportsTheValueItReplaced(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		replaced, wasHeld, err := words.Put(tx, "a", "alpha")
		if err != nil {
			return wrap(err)
		}
		if wasHeld {
			t.Fatalf("first Put replaced %q", replaced)
		}

		replaced, wasHeld, err = words.Put(tx, "a", "again")
		if err != nil {
			return wrap(err)
		}
		if !wasHeld || replaced != "alpha" {
			t.Fatalf("second Put replaced %q, %v, want alpha, true", replaced, wasHeld)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDeleteReportsTheValueItRemoved(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if _, _, err := words.Put(tx, "a", "alpha"); err != nil {
			return wrap(err)
		}

		removed, wasHeld, err := words.Delete(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if !wasHeld || removed != "alpha" {
			t.Fatalf("Delete removed %q, %v, want alpha, true", removed, wasHeld)
		}

		removed, wasHeld, err = words.Delete(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if wasHeld {
			t.Fatalf("second Delete removed %q", removed)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}
