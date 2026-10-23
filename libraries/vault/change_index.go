package vault

import (
	"bytes"
	"fmt"
	"hash/fnv"
)

type ChangeIndex[R, E any] struct {
	projectionRegistration
	entries KeyLayout[E]
	entryOf func(R) E
}

type Change[E any] struct {
	Entry             E
	storedFingerprint string
}

func (v *Vault) RegisterChangeIndex[R, E any](
	bucket Name,
	version string,
	entries KeyLayout[E],
	entryOf func(R) E,
) (*ChangeIndex[R, E], error) {
	if err := v.provisionRecordsBucket(bucket); err != nil {
		return nil, err
	}

	return &ChangeIndex[R, E]{
		projectionRegistration: projectionRegistration{bucket: bucket, version: version},
		entries:                entries,
		entryOf:                entryOf,
	}, nil
}

func (i *ChangeIndex[R, E]) Scan(
	tx *Txn,
	entries KeyRange,
	fn func(Change[E]) (bool, error),
) error {
	if err := i.refuseUnattached(); err != nil {
		return err
	}
	if err := tx.etx.Records(i.bucket).Scan(
		entries,
		func(key, storedFingerprint []byte) (bool, error) {
			entry, err := i.entries.Decode(key)
			if err != nil {
				return false, fmt.Errorf("decode %s entry: %w", i.bucket, err)
			}

			return fn(Change[E]{Entry: entry, storedFingerprint: string(storedFingerprint)})
		},
	); err != nil {
		return fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) Acknowledge(
	tx *Txn,
	change Change[E],
) (wasAcknowledged bool, err error) {
	if err := i.refuseUnattached(); err != nil {
		return false, err
	}
	if !tx.etx.Writable() {
		return false, errReadOnly
	}
	tx.calledWriteOperation = true

	entryKey := i.entries.Encode(change.Entry).encoded
	storedFingerprint, err := tx.etx.Records(i.bucket).Get(entryKey)
	if err != nil {
		return false, fmt.Errorf("read %s entry: %w", i.bucket, err)
	}
	if storedFingerprint == nil || string(storedFingerprint) != change.storedFingerprint {
		return false, nil
	}
	if err := tx.etx.Records(i.bucket).DeleteHeld(entryKey, storedFingerprint); err != nil {
		return false, fmt.Errorf("acknowledge %s entry: %w", i.bucket, err)
	}

	return true, nil
}

func (i *ChangeIndex[R, E]) Len(tx *Txn) (int, error) {
	if err := i.refuseUnattached(); err != nil {
		return 0, err
	}

	return lengthOf(tx, i.bucket)
}

func (i *ChangeIndex[R, E]) recordAdded(tx *Txn, added storedValue[R]) error {
	if err := tx.etx.Records(i.bucket).PutAbsent(
		i.entryKeyOf(added.value),
		storedFingerprintOf(added.encoded),
	); err != nil {
		return fmt.Errorf("add %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) entryKeyOf(record R) []byte {
	return i.entries.Encode(i.entryOf(record)).encoded
}

func storedFingerprintOf(storedRecord []byte) []byte {
	fingerprint := fnv.New64a()
	_, _ = fingerprint.Write(storedRecord)

	return recordFrom(fingerprint.Sum(nil))
}

func (i *ChangeIndex[R, E]) recordRemoved(tx *Txn, removed storedValue[R]) error {
	return i.removeEntry(tx, i.entryKeyOf(removed.value))
}

func (i *ChangeIndex[R, E]) removeEntry(tx *Txn, entryKey []byte) error {
	if _, err := tx.etx.Records(i.bucket).Delete(entryKey); err != nil {
		return fmt.Errorf("remove %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) recordReplaced(tx *Txn, previous, current storedValue[R]) error {
	previousKey, currentKey := i.entryKeyOf(previous.value), i.entryKeyOf(current.value)
	if !bytes.Equal(previousKey, currentKey) {
		if err := i.removeEntry(tx, previousKey); err != nil {
			return err
		}
	}
	if _, err := tx.etx.Records(i.bucket).Put(
		currentKey,
		storedFingerprintOf(current.encoded),
	); err != nil {
		return fmt.Errorf("replace %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
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
