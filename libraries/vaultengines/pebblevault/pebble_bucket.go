package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleBucket struct {
	entries storedEntries
	tally   storedBucketTally
	staged  *pebble.Batch
}

func (b pebbleBucket) Get(key []byte) ([]byte, error) {
	return b.entries.valueAt(key)
}

func (b pebbleBucket) Put(key []byte, newValue []byte) error {
	previousValue, err := b.entries.valueAt(key)
	if err != nil {
		return err
	}
	if err := b.staged.Set(
		b.entries.region.absoluteKeyFrom(key),
		newValue,
		pebble.NoSync,
	); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if previousValue == nil {
		return b.tally.adjustBy(1, int64(len(key)+len(newValue)))
	}

	return b.tally.adjustBy(0, int64(len(newValue)-len(previousValue)))
}

func (b pebbleBucket) Delete(key []byte) (bool, error) {
	previousValue, err := b.entries.valueAt(key)
	if err != nil {
		return false, err
	}
	if previousValue == nil {
		return false, nil
	}
	if err := b.staged.Delete(b.entries.region.absoluteKeyFrom(key), pebble.NoSync); err != nil {
		return false, fmt.Errorf("delete: %w", err)
	}
	if err := b.tally.adjustBy(-1, -int64(len(key)+len(previousValue))); err != nil {
		return false, err
	}

	return true, nil
}

func (b pebbleBucket) Len() (int, error) {
	tally, err := b.tally.value()

	return tally.entries, err
}

func (b pebbleBucket) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	return b.entries.visit(keys, fn)
}
