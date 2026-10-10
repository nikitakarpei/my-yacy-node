package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleAmounts struct {
	entries        storedEntries
	changes        *pebble.Batch
	loweredAmounts *amountKeys
}

func (a pebbleAmounts) Get(key []byte) (int, error) {
	raw, err := a.entries.valueAt(key)
	if err != nil || raw == nil {
		return 0, err
	}

	return countFrom(raw)
}

func countFrom(raw []byte) (int, error) {
	amount, err := amountFrom(raw)
	if err != nil {
		return 0, err
	}

	return int(max(amount, 0)), nil
}

func (a pebbleAmounts) Raise(key []byte, by uint) error {
	//nolint:gosec // G115: an amount changes by a count of things, far below MaxInt64.
	return a.stageChange(key, int64(by))
}

func (a pebbleAmounts) stageChange(key []byte, change int64) error {
	if err := a.changes.Merge(
		a.entries.region.absoluteKeyFrom(key),
		encodedAmount(change),
		pebble.NoSync,
	); err != nil {
		return fmt.Errorf("store amount: %w", err)
	}

	return nil
}

func (a pebbleAmounts) Lower(key []byte, by uint) error {
	//nolint:gosec // G115: an amount changes by a count of things, far below MaxInt64.
	if err := a.stageChange(key, -int64(by)); err != nil {
		return err
	}
	a.loweredAmounts.add(a.entries.region.absoluteKeyFrom(key))

	return nil
}

func (a pebbleAmounts) Scan(
	keys vault.KeyRange,
	fn func(key []byte, amount int) (bool, error),
) error {
	return a.entries.visit(keys, func(key, raw []byte) (bool, error) {
		amount, err := countFrom(raw)
		if err != nil || amount == 0 {
			return err == nil, err
		}

		return fn(key, amount)
	})
}

type amountKeys struct {
	keys [][]byte
}

func (k *amountKeys) add(key []byte) {
	k.keys = append(k.keys, key)
}
