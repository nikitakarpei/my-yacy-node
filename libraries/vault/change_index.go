package vault

import (
	"bytes"
	"fmt"
)

type ChangeIndex[R, E any] struct {
	projectionRegistration
	entries KeyLayout[E]
	entryOf func(R) E
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
	fn func(E) (bool, error),
) error {
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

func (i *ChangeIndex[R, E]) Acknowledge(
	tx *Txn,
	entry E,
) (wasAcknowledged bool, err error) {
	if err := i.refuseUnfollowed(); err != nil {
		return false, err
	}
	if !tx.etx.Writable() {
		return false, errReadOnly
	}
	tx.calledWriteOperation = true

	acknowledgedRecord, err := tx.etx.Records(i.bucket).Delete(i.entries.Encode(entry).encoded)
	if err != nil {
		return false, fmt.Errorf("acknowledge %s entry: %w", i.bucket, err)
	}

	return acknowledgedRecord != nil, nil
}

func (i *ChangeIndex[R, E]) Len(tx *Txn) (int, error) {
	if err := i.refuseUnfollowed(); err != nil {
		return 0, err
	}

	return lengthOf(tx, i.bucket)
}

func (i *ChangeIndex[R, E]) recordAdded(tx *Txn, record storedForm[R]) error {
	if err := tx.etx.Records(i.bucket).
		PutAbsent(i.entryKeyOf(record.form), recordFrom(nil)); err != nil {
		return fmt.Errorf("add %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) entryKeyOf(record R) []byte {
	return i.entries.Encode(i.entryOf(record)).encoded
}

func (i *ChangeIndex[R, E]) recordRemoved(tx *Txn, record storedForm[R]) error {
	return i.removeEntry(tx, i.entryKeyOf(record.form))
}

func (i *ChangeIndex[R, E]) removeEntry(tx *Txn, entryKey []byte) error {
	if _, err := tx.etx.Records(i.bucket).Delete(entryKey); err != nil {
		return fmt.Errorf("remove %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *ChangeIndex[R, E]) recordReplaced(tx *Txn, previous, current storedForm[R]) error {
	previousKey, currentKey := i.entryKeyOf(previous.form), i.entryKeyOf(current.form)
	if !bytes.Equal(previousKey, currentKey) {
		if err := i.removeEntry(tx, previousKey); err != nil {
			return err
		}
	}
	if _, err := tx.etx.Records(i.bucket).Put(currentKey, recordFrom(nil)); err != nil {
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
