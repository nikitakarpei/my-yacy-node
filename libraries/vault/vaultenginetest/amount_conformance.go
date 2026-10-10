package vaultenginetest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func runAmountConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"AmountsAddUpAcrossTransactions",
		func(t *testing.T) { amountsAddUpAcrossTransactions(t, open) },
	)
	t.Run(
		"TransactionReadsItsOwnAmountChanges",
		func(t *testing.T) { transactionReadsItsOwnAmountChanges(t, open) },
	)
	t.Run(
		"AmountLoweredToZeroIsAbsent",
		func(t *testing.T) { amountLoweredToZeroIsAbsent(t, open) },
	)
	t.Run(
		"AmountScanVisitsRangeInOrder",
		func(t *testing.T) { amountScanVisitsRangeInOrder(t, open) },
	)
	t.Run(
		"AbortedTransactionLeavesAmountsUnchanged",
		func(t *testing.T) { abortedTransactionLeavesAmountsUnchanged(t, open) },
	)
	t.Run(
		"ConcurrentRaisesKeepEveryRaise",
		func(t *testing.T) { concurrentRaisesKeepEveryRaise(t, open) },
	)
	t.Run(
		"RepeatedWriteRaisesOneTime",
		func(t *testing.T) { repeatedWriteRaisesOneTime(t, open) },
	)
}

var errAbortedOnPurpose = errors.New("aborted on purpose")

const talliesBucket vault.Name = "tallies"

func registerTallies(t *testing.T, v *vault.Vault) *vault.Amounts[string] {
	t.Helper()

	tallies, err := v.RegisterAmounts(talliesBucket, stringKeyLayout)
	if err != nil {
		t.Fatalf("RegisterAmounts: %v", err)
	}

	return tallies
}

func amountsAddUpAcrossTransactions(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)

	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 5) })
	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Lower(tx, "a", 2) })
	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 4) })

	if amount := storedAmountOf(t, v, tallies, "a"); amount != 7 {
		t.Fatalf("amount of a = %d, want 7", amount)
	}
}

func changeAmounts(t *testing.T, v *vault.Vault, change func(*vault.Txn) error) {
	t.Helper()

	if err := v.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func storedAmountOf(t *testing.T, v *vault.Vault, amounts *vault.Amounts[string], key string) int {
	t.Helper()

	var stored int
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		amount, err := amounts.Get(tx, key)
		stored = amount

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return stored
}

func transactionReadsItsOwnAmountChanges(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)
	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 5) })

	changeAmounts(t, v, func(tx *vault.Txn) error {
		if err := tallies.Raise(tx, "a", 3); err != nil {
			return wrapTest(err)
		}
		amount, err := tallies.Get(tx, "a")
		if err != nil {
			return wrapTest(err)
		}
		if amount != 8 {
			t.Fatalf("amount inside the transaction = %d, want 8", amount)
		}

		return nil
	})
}

func amountLoweredToZeroIsAbsent(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)
	changeAmounts(t, v, func(tx *vault.Txn) error {
		return errors.Join(tallies.Raise(tx, "a", 2), tallies.Raise(tx, "b", 1))
	})

	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Lower(tx, "a", 2) })

	if amount := storedAmountOf(t, v, tallies, "a"); amount != 0 {
		t.Fatalf("amount of a = %d, want 0", amount)
	}
	if scanned := scannedAmounts(
		t,
		v,
		tallies,
		vault.EveryKey(),
	); !slices.Equal(
		scanned,
		[]string{"b"},
	) {
		t.Fatalf("scanned amounts = %v, want [b]", scanned)
	}

	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 1) })
	if amount := storedAmountOf(t, v, tallies, "a"); amount != 1 {
		t.Fatalf("amount of a raised again = %d, want 1", amount)
	}
}

func scannedAmounts(
	t *testing.T,
	v *vault.Vault,
	amounts *vault.Amounts[string],
	keys vault.KeyRange,
) []string {
	t.Helper()

	var scanned []string
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		return amounts.Scan(tx, keys, func(key string, _ int) (bool, error) {
			scanned = append(scanned, key)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return scanned
}

func amountScanVisitsRangeInOrder(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)
	changeAmounts(t, v, func(tx *vault.Txn) error {
		return errors.Join(
			tallies.Raise(tx, "d", 1),
			tallies.Raise(tx, "b", 1),
			tallies.Raise(tx, "a", 1),
			tallies.Raise(tx, "c", 1),
		)
	})

	scanned := scannedAmounts(t, v, tallies, stringKeyParts.KeysFromFirst("b"))
	if !slices.Equal(scanned, []string{"b", "c", "d"}) {
		t.Fatalf("scanned amounts = %v, want [b c d]", scanned)
	}
}

func abortedTransactionLeavesAmountsUnchanged(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)
	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 2) })

	err := v.Update(context.Background(), func(tx *vault.Txn) error {
		if err := tallies.Lower(tx, "a", 2); err != nil {
			return wrapTest(err)
		}

		return errAbortedOnPurpose
	})
	if !errors.Is(err, errAbortedOnPurpose) {
		t.Fatalf("Update = %v, want the closure failure", err)
	}

	if amount := storedAmountOf(t, v, tallies, "a"); amount != 2 {
		t.Fatalf("amount of a after the abort = %d, want 2", amount)
	}
}

func concurrentRaisesKeepEveryRaise(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	tallies := registerTallies(t, v)

	failures := make(chan error, concurrentWriters)
	var writing sync.WaitGroup
	for range concurrentWriters {
		writing.Go(func() {
			for range incrementsPerWriter {
				if err := v.Update(context.Background(), func(tx *vault.Txn) error {
					return tallies.Raise(tx, counterKey, 1)
				}); err != nil {
					failures <- err

					return
				}
			}
		})
	}
	writing.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("Update: %v", err)
	}

	if amount := storedAmountOf(
		t,
		v,
		tallies,
		counterKey,
	); amount != concurrentWriters*incrementsPerWriter {
		t.Fatalf("amount = %d, want %d", amount, concurrentWriters*incrementsPerWriter)
	}
}

func repeatedWriteRaisesOneTime(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openRepeatingVault(t, open)
	tallies := registerTallies(t, v)

	changeAmounts(t, v, func(tx *vault.Txn) error { return tallies.Raise(tx, "a", 1) })

	if amount := storedAmountOf(t, v, tallies, "a"); amount != 1 {
		t.Fatalf("amount of a = %d, want 1", amount)
	}
}
