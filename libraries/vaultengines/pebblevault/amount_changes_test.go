package pebblevault_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func registerTallies(t *testing.T, store *vault.Vault) *vault.Amounts[string] {
	t.Helper()

	tallies, err := store.RegisterAmounts(vault.Name("tallies"), stringKeyLayout)
	if err != nil {
		t.Fatalf("RegisterAmounts: %v", err)
	}

	return tallies
}

func change(t *testing.T, store *vault.Vault, fn func(tx *vault.Txn) error) {
	t.Helper()

	if err := store.Update(context.Background(), fn); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func amountsWithin(
	t *testing.T,
	tx *vault.Txn,
	tallies *vault.Amounts[string],
	keys vault.KeyRange,
) map[string]int {
	t.Helper()

	scanned := map[string]int{}
	if err := tallies.Scan(tx, keys, func(key string, amount int) (bool, error) {
		scanned[key] = amount

		return true, nil
	}); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	return scanned
}

func committedAmountsOf(
	t *testing.T,
	store *vault.Vault,
	tallies *vault.Amounts[string],
) map[string]int {
	t.Helper()

	var committed map[string]int
	if err := store.View(context.Background(), func(tx *vault.Txn) error {
		committed = amountsWithin(t, tx, tallies, vault.EveryKey())

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return committed
}

func amountOf(t *testing.T, tx *vault.Txn, tallies *vault.Amounts[string], key string) int {
	t.Helper()

	amount, err := tallies.Get(tx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	return amount
}

func TestManyAmountChangesInOneTransactionAddUp(t *testing.T) {
	store := openVault(t, 0)
	tallies := registerTallies(t, store)

	change(t, store, func(tx *vault.Txn) error {
		for range 100 {
			if err := tallies.Raise(tx, "a", 3); err != nil {
				return err
			}
			if err := tallies.Lower(tx, "a", 1); err != nil {
				return err
			}
		}

		return tallies.Raise(tx, "b", 5)
	})

	want := map[string]int{"a": 200, "b": 5}
	if committed := committedAmountsOf(t, store, tallies); !maps.Equal(committed, want) {
		t.Fatalf("committed amounts = %v, want %v", committed, want)
	}
}

func TestTransactionReadsItsAmountChangesBeforeTheyAreStored(t *testing.T) {
	store := openVault(t, 0)
	tallies := registerTallies(t, store)
	change(t, store, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 2) })

	change(t, store, func(tx *vault.Txn) error {
		if err := tallies.Raise(tx, "a", 3); err != nil {
			return err
		}
		if err := tallies.Raise(tx, "z", 4); err != nil {
			return err
		}
		if amounts := []int{
			amountOf(t, tx, tallies, "a"),
			amountOf(t, tx, tallies, "z"),
		}; !slices.Equal(
			amounts,
			[]int{5, 4},
		) {
			t.Fatalf("amounts of a and z = %v, want [5 4]", amounts)
		}
		want := map[string]int{"z": 4}
		if scanned := amountsWithin(t, tx, tallies, stringKeyParts.KeysFromFirst("m")); !maps.Equal(
			scanned,
			want,
		) {
			t.Fatalf("scanned amounts from m = %v, want %v", scanned, want)
		}
		if err := tallies.Raise(tx, "z", 1); err != nil {
			return err
		}
		want = map[string]int{"a": 5, "z": 5}
		if scanned := amountsWithin(t, tx, tallies, vault.EveryKey()); !maps.Equal(scanned, want) {
			t.Fatalf("scanned amounts = %v, want %v", scanned, want)
		}

		return nil
	})
}

func TestAmountLoweredToZeroInOneTransactionLeavesTheScan(t *testing.T) {
	store := openVault(t, 0)
	tallies := registerTallies(t, store)
	change(t, store, func(tx *vault.Txn) error {
		return errors.Join(tallies.Raise(tx, "a", 3), tallies.Raise(tx, "b", 1))
	})

	want := map[string]int{"b": 1}
	change(t, store, func(tx *vault.Txn) error {
		for range 3 {
			if err := tallies.Lower(tx, "a", 1); err != nil {
				return err
			}
		}
		if err := tallies.Raise(tx, "c", 2); err != nil {
			return err
		}
		if err := tallies.Lower(tx, "c", 2); err != nil {
			return err
		}
		if scanned := amountsWithin(t, tx, tallies, vault.EveryKey()); !maps.Equal(scanned, want) {
			t.Fatalf("scanned amounts inside the transaction = %v, want %v", scanned, want)
		}

		return nil
	})

	if committed := committedAmountsOf(t, store, tallies); !maps.Equal(committed, want) {
		t.Fatalf("committed amounts = %v, want %v", committed, want)
	}
}

func TestRecordStoredAndDeletedInOneTransactionLeavesTheBucketAsBefore(t *testing.T) {
	store := openVault(t, 0)
	words := registerWords(t, store)
	change(t, store, func(tx *vault.Txn) error {
		_, err := words.Put(tx, "a", "alpha")

		return err
	})

	growth := usedBytesGrowthOf(t, store, func(tx *vault.Txn) error {
		if _, err := words.Put(tx, "b", "beta"); err != nil {
			return err
		}
		_, err := words.Delete(tx, "b")

		return err
	})

	if growth != 0 {
		t.Fatalf("UsedBytes changed by %d, want 0", growth)
	}
	if err := store.View(context.Background(), func(tx *vault.Txn) error {
		length, err := words.Len(tx)
		if length != 1 {
			t.Fatalf("Len = %d, want 1", length)
		}

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}
