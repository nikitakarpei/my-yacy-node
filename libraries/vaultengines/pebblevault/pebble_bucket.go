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

func (b pebbleBucket) Put(key []byte, newValue []byte) ([]byte, error) {
	previousValue, err := b.entries.valueAt(key)
	if err != nil {
		return nil, err
	}
	if err := b.staged.Set(
		b.entries.region.absoluteKeyFrom(key),
		newValue,
		pebble.NoSync,
	); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	if previousValue == nil {
		return nil, b.tally.adjustBy(1, int64(len(key)+len(newValue)))
	}

	return previousValue, b.tally.adjustBy(0, int64(len(newValue)-len(previousValue)))
}

func (b pebbleBucket) Delete(key []byte) ([]byte, error) {
	previousValue, err := b.entries.valueAt(key)
	if err != nil {
		return nil, err
	}
	if previousValue == nil {
		return nil, nil
	}
	if err := b.staged.Delete(b.entries.region.absoluteKeyFrom(key), pebble.NoSync); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}
	if err := b.tally.adjustBy(-1, -int64(len(key)+len(previousValue))); err != nil {
		return nil, err
	}

	return previousValue, nil
}

func (b pebbleBucket) Len() (int, error) {
	tally, err := b.tally.value()

	return tally.entries, err
}

func (b pebbleBucket) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	return b.entries.visit(keys, fn)
}
