package pebblevault

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleRecords struct {
	entries storedEntries
	tally   storedBucketTally
}

func (b pebbleRecords) Get(key []byte) ([]byte, error) {
	return b.entries.valueAt(key)
}

func (b pebbleRecords) Put(key []byte, record []byte) ([]byte, error) {
	replacedRecord, err := b.entries.store(key, record)
	if err != nil {
		return nil, err
	}
	if replacedRecord == nil {
		b.tally.adjustBy(1, int64(len(key)+len(record)))

		return nil, nil
	}
	b.tally.adjustBy(0, int64(len(record)-len(replacedRecord)))

	return replacedRecord, nil
}

func (b pebbleRecords) Delete(key []byte) ([]byte, error) {
	deletedRecord, err := b.entries.remove(key)
	if err != nil || deletedRecord == nil {
		return nil, err
	}
	b.tally.adjustBy(-1, -int64(len(key)+len(deletedRecord)))

	return deletedRecord, nil
}

func (b pebbleRecords) PutAbsent(key []byte, record []byte) error {
	if err := b.entries.storeAbsent(key, record); err != nil {
		return err
	}
	b.tally.adjustBy(1, int64(len(key)+len(record)))

	return nil
}

func (b pebbleRecords) DeleteHeld(key []byte, heldRecord []byte) error {
	if err := b.entries.removeHeld(key); err != nil {
		return err
	}
	b.tally.adjustBy(-1, -int64(len(key)+len(heldRecord)))

	return nil
}

func (b pebbleRecords) Len() (int, error) {
	tally, err := b.tally.value()

	return tally.records, err
}

func (b pebbleRecords) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	return b.entries.visit(keys, fn)
}
