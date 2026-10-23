package pebblevault

import (
	"bytes"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleAmounts struct {
	entries storedEntries
	changes *amountChanges
	scans   *scanTally
}

func (a pebbleAmounts) Get(key []byte) (int, error) {
	absoluteKey := a.entries.absoluteKeyFrom(key)
	stored, err := a.storedAmountAt(absoluteKey)

	return countFrom(stored + a.changes.changeAt(absoluteKey)), err
}

func (a pebbleAmounts) storedAmountAt(absoluteKey []byte) (int64, error) {
	raw, err := a.entries.valueAtAbsoluteKey(absoluteKey)
	if err != nil || raw == nil {
		return 0, err
	}

	return amountFrom(raw)
}

func countFrom(amount int64) int {
	return int(max(amount, 0))
}

func (a pebbleAmounts) Raise(key []byte, by uint) error {
	//nolint:gosec // G115: an amount changes by a count of things, far below MaxInt64.
	a.change(key, int64(by))

	return nil
}

func (a pebbleAmounts) change(key []byte, by int64) {
	a.changes.add(a.entries.absoluteKeyFrom(key), by)
}

func (a pebbleAmounts) Lower(key []byte, by uint) error {
	//nolint:gosec // G115: an amount changes by a count of things, far below MaxInt64.
	a.change(key, -int64(by))

	return nil
}

func (a pebbleAmounts) DeleteWithin(keys vault.KeyRange) error {
	a.changes.dropWithin(a.entries.boundsFor(keys))

	var storedKeys [][]byte
	if err := a.entries.visit(keys, func(key, _ []byte) (bool, error) {
		storedKeys = append(storedKeys, bytes.Clone(key))

		return true, nil
	}); err != nil {
		return err
	}
	for _, key := range storedKeys {
		if err := a.entries.removeHeld(key); err != nil {
			return err
		}
	}

	return nil
}

func (a pebbleAmounts) Scan(
	keys vault.KeyRange,
	fn func(key []byte, amount int) (bool, error),
) error {
	if err := a.changes.stageWithin(a.entries.boundsFor(keys)); err != nil {
		return err
	}

	return a.entries.visit(keys, func(key, raw []byte) (bool, error) {
		amount, err := amountFrom(raw)
		if err != nil {
			return false, err
		}
		a.scans.count(amount)
		if countFrom(amount) == 0 {
			return true, nil
		}

		return fn(key, countFrom(amount))
	})
}
