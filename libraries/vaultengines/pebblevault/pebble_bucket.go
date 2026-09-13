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

func (b pebbleBucket) Put(key []byte, record []byte) ([]byte, error) {
	replacedRecord, err := b.entries.valueAt(key)
	if err != nil {
		return nil, err
	}
	if err := b.staged.Set(
		b.entries.region.absoluteKeyFrom(key),
		record,
		pebble.NoSync,
	); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	if replacedRecord == nil {
		return nil, b.tally.adjustBy(1, int64(len(key)+len(record)))
	}

	return replacedRecord, b.tally.adjustBy(0, int64(len(record)-len(replacedRecord)))
}

func (b pebbleBucket) Delete(key []byte) ([]byte, error) {
	deletedRecord, err := b.entries.valueAt(key)
	if err != nil {
		return nil, err
	}
	if deletedRecord == nil {
		return nil, nil
	}
	if err := b.staged.Delete(b.entries.region.absoluteKeyFrom(key), pebble.NoSync); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}
	if err := b.tally.adjustBy(-1, -int64(len(key)+len(deletedRecord))); err != nil {
		return nil, err
	}

	return deletedRecord, nil
}

func (b pebbleBucket) Len() (int, error) {
	tally, err := b.tally.value()

	return tally.entries, err
}

func (b pebbleBucket) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	return b.entries.visit(keys, fn)
}
