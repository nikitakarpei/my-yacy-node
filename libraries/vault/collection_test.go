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
		stringKeyLayout,
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
	v, words := openUndecodableWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	getErr := v.View(ctx, func(tx *vault.Txn) error {
		_, _, err := words.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}

		return nil
	})
	if getErr == nil {
		t.Fatal("Get with failing decode succeeded, want error")
	}

	scanErr := v.View(ctx, func(tx *vault.Txn) error {
		return words.Scan(
			tx,
			vault.EveryKey(),
			func(string, string) (bool, error) { return true, nil },
		)
	})
	if scanErr == nil {
		t.Fatal("Scan with failing decode succeeded, want error")
	}
}

func TestPutReturningReportsTheValueItReplaced(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		replacedValue, wasReplaced, err := words.PutReturning(tx, "a", "alpha")
		if err != nil {
			return wrap(err)
		}
		if wasReplaced {
			t.Fatalf("first PutReturning replaced %q", replacedValue)
		}

		replacedValue, wasReplaced, err = words.PutReturning(tx, "a", "again")
		if err != nil {
			return wrap(err)
		}
		if !wasReplaced || replacedValue != "alpha" {
			t.Fatalf(
				"second PutReturning replaced %q, %v, want alpha, true",
				replacedValue,
				wasReplaced,
			)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDeleteReturningReportsTheValueItRemoved(t *testing.T) {
	ctx := context.Background()
	v, words := openWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if err := words.Put(tx, "a", "alpha"); err != nil {
			return wrap(err)
		}

		deletedValue, wasDeleted, err := words.DeleteReturning(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if !wasDeleted || deletedValue != "alpha" {
			t.Fatalf("DeleteReturning removed %q, %v, want alpha, true", deletedValue, wasDeleted)
		}

		deletedValue, wasDeleted, err = words.DeleteReturning(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if wasDeleted {
			t.Fatalf("second DeleteReturning removed %q", deletedValue)
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestReturningWritesFailOverARecordThatDoesNotDecode(t *testing.T) {
	ctx := context.Background()
	v, words := openUndecodableWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	putErr := v.Update(ctx, func(tx *vault.Txn) error {
		_, _, err := words.PutReturning(tx, "a", "again")

		return err
	})
	if putErr == nil {
		t.Fatal("PutReturning over a record that does not decode succeeded, want error")
	}

	deleteErr := v.Update(ctx, func(tx *vault.Txn) error {
		_, _, err := words.DeleteReturning(tx, "a")

		return err
	})
	if deleteErr == nil {
		t.Fatal("DeleteReturning over a record that does not decode succeeded, want error")
	}
}

func TestPlainWritesSucceedOverARecordThatDoesNotDecode(t *testing.T) {
	ctx := context.Background()
	v, words := openUndecodableWords(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "again")
	}); err != nil {
		t.Fatalf("Put over a record that does not decode: %v", err)
	}

	var wasDeleted bool
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		removed, err := words.Delete(tx, "a")
		wasDeleted = removed

		return err
	}); err != nil {
		t.Fatalf("Delete over a record that does not decode: %v", err)
	}
	if !wasDeleted {
		t.Fatal("Delete reported nothing deleted, want the record removed")
	}
}
