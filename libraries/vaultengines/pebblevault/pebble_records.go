package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleRecords struct {
	entries storedEntries
	tally   storedBucketTally
	staged  *pebble.Batch
}

func (b pebbleRecords) Get(key []byte) ([]byte, error) {
	return b.entries.valueAt(key)
}

func (b pebbleRecords) Put(key []byte, record []byte) ([]byte, error) {
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

func (b pebbleRecords) Delete(key []byte) ([]byte, error) {
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

func (b pebbleRecords) Len() (int, error) {
	tally, err := b.tally.value()

	return tally.records, err
}

func (b pebbleRecords) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	return b.entries.visit(keys, fn)
}
