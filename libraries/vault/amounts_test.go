package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func openTallies(t *testing.T) (*vault.Vault, *vault.Amounts[string]) {
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

	tallies, err := v.RegisterAmounts(vault.Name("tallies"), stringKeyLayout)
	if err != nil {
		t.Fatalf("RegisterAmounts: %v", err)
	}

	return v, tallies
}

func TestRaisedAndLoweredAmountsAddUp(t *testing.T) {
	ctx := context.Background()
	v, tallies := openTallies(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return errors.Join(
			tallies.Raise(tx, "a", 5),
			tallies.Lower(tx, "a", 2),
			tallies.Raise(tx, "b", 1),
		)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		amount, err := tallies.Get(tx, "a")
		if err != nil {
			return wrap(err)
		}
		if amount != 3 {
			t.Fatalf("Get(a) = %d, want 3", amount)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func TestReadOnlyTransactionRefusesToChangeAmounts(t *testing.T) {
	v, tallies := openTallies(t)

	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		return errors.Join(tallies.Raise(tx, "a", 1), tallies.Lower(tx, "a", 1))
	}); err == nil {
		t.Fatal("View allowed an amount to change")
	}
}

func TestAmountsAndCollectionsShareOneNameSpace(t *testing.T) {
	v, _ := openTallies(t)

	if _, err := v.RegisterCollection(
		vault.Name("tallies"),
		stringKeyLayout,
		stringValueCodec{},
	); err == nil {
		t.Fatal("RegisterCollection accepted the name of registered amounts")
	}
}

func TestRecordCountsByBucketLeavesAmountsOut(t *testing.T) {
	ctx := context.Background()
	v, tallies := openTallies(t)

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return tallies.Raise(tx, "a", 1)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	entries, err := v.RecordCountsByBucket(ctx)
	if err != nil {
		t.Fatalf("RecordCountsByBucket: %v", err)
	}
	if _, listed := entries[vault.Name("tallies")]; listed {
		t.Fatalf("RecordCountsByBucket = %v, want no amounts", entries)
	}
}
