package vault

import (
	"bytes"
	"fmt"
)

type ValueIndex[R, E any] struct {
	projectionRegistration
	entries KeyLayout[E]
	entryOf func(R) E
}

func (v *Vault) RegisterValueIndex[R, E any](
	bucket Name,
	version string,
	entries KeyLayout[E],
	entryOf func(R) E,
) (*ValueIndex[R, E], error) {
	if err := v.provisionRecordsBucket(bucket); err != nil {
		return nil, err
	}

	return &ValueIndex[R, E]{
		projectionRegistration: projectionRegistration{bucket: bucket, version: version},
		entries:                entries,
		entryOf:                entryOf,
	}, nil
}

func (i *ValueIndex[R, E]) Scan(tx *Txn, entries KeyRange, fn func(E) (bool, error)) error {
	if err := i.refuseUnfollowed(); err != nil {
		return err
	}
	if err := tx.etx.Records(i.bucket).Scan(entries, func(key, _ []byte) (bool, error) {
		entry, err := i.entries.Decode(key)
		if err != nil {
			return false, fmt.Errorf("decode %s entry: %w", i.bucket, err)
		}

		return fn(entry)
	}); err != nil {
		return fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return nil
}

func (i *ValueIndex[R, E]) recordAdded(tx *Txn, record R) error {
	return i.addEntry(tx, i.entryKeyOf(record))
}

func (i *ValueIndex[R, E]) entryKeyOf(record R) []byte {
	return i.entries.Encode(i.entryOf(record)).encoded
}

func (i *ValueIndex[R, E]) addEntry(tx *Txn, entryKey []byte) error {
	if err := tx.etx.Records(i.bucket).PutAbsent(entryKey, recordFrom(nil)); err != nil {
		return fmt.Errorf("add %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ValueIndex[R, E]) recordRemoved(tx *Txn, record R) error {
	return i.removeEntry(tx, i.entryKeyOf(record))
}

func (i *ValueIndex[R, E]) removeEntry(tx *Txn, entryKey []byte) error {
	if err := tx.etx.Records(i.bucket).DeleteHeld(entryKey, recordFrom(nil)); err != nil {
		return fmt.Errorf("remove %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ValueIndex[R, E]) recordReplaced(tx *Txn, previous, current R) error {
	previousKey, currentKey := i.entryKeyOf(previous), i.entryKeyOf(current)
	if bytes.Equal(previousKey, currentKey) {
		return nil
	}
	if err := i.removeEntry(tx, previousKey); err != nil {
		return err
	}

	return i.addEntry(tx, currentKey)
}

func (i *ValueIndex[R, E]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	entryKeys, nextKey, err := recordKeysFrom(tx, i.bucket, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for _, entryKey := range entryKeys {
		if err := i.removeEntry(tx, entryKey); err != nil {
			return nil, err
		}
	}

	return nextKey, nil
}

func recordKeysFrom(
	tx *Txn,
	bucket Name,
	firstKey []byte,
	limit int,
) (keys [][]byte, nextKey []byte, err error) {
	if err := tx.etx.Records(bucket).Scan(
		KeyRange{firstIncluded: firstKey},
		func(key, _ []byte) (bool, error) {
			if len(keys) == limit {
				nextKey = bytes.Clone(key)

				return false, nil
			}
			keys = append(keys, bytes.Clone(key))

			return true, nil
		},
	); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", bucket, err)
	}

	return keys, nextKey, nil
}
