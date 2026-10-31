package vault

import (
	"bytes"
	"context"
	"errors"
	"fmt"
)

var errReadOnly = errors.New("write inside read-only transaction")

type Collection[K, V any] struct {
	name        Name
	codec       recordCodec[K, V]
	projections attachedProjections[K, V]
}

func newCollection[K, V any](
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
	projections Projections[K, V],
) *Collection[K, V] {
	codec := recordCodec[K, V]{keys: keys, values: values}

	return &Collection[K, V]{
		name:        bucket,
		codec:       codec,
		projections: attachedProjections[K, V]{projections: projections, codec: codec},
	}
}

func (v *Vault) RegisterCollection[K, V any](
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
) (*Collection[K, V], error) {
	if err := v.provisionRecordsBucket(bucket); err != nil {
		return nil, err
	}

	return newCollection(bucket, keys, values, nil), nil
}

func (v *Vault) RegisterProjectedCollection[K, V any](
	ctx context.Context,
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
	projections Projections[K, V],
) (*Collection[K, V], error) {
	return v.registerProjected(ctx, newCollection(bucket, keys, values, projections))
}

func (v *Vault) registerProjected[K, V any](
	ctx context.Context,
	collection *Collection[K, V],
) (*Collection[K, V], error) {
	if err := v.provisionRecordsBucket(collection.name); err != nil {
		return nil, err
	}
	if err := v.attachProjections(
		ctx,
		collection.name,
		collection.projections,
		func(tx *Txn) storedRecords { return tx.etx.Records(collection.name) },
	); err != nil {
		return nil, err
	}

	return collection, nil
}

func (c *Collection[K, V]) Get(tx *Txn, key K) (V, bool, error) {
	var zero V

	record, err := tx.etx.Records(c.name).Get(c.codec.encodedKeyOf(key))
	if err != nil {
		return zero, false, fmt.Errorf("read %s: %w", c.name, err)
	}
	if record == nil {
		return zero, false, nil
	}

	val, err := c.codec.valueFrom(record)
	if err != nil {
		return zero, false, fmt.Errorf("read %s: %w", c.name, err)
	}

	return val, true, nil
}

func (c *Collection[K, V]) Put(tx *Txn, key K, val V) (wasReplaced bool, err error) {
	replacedRecord, err := c.storeRecord(tx, key, val)
	if err != nil {
		return false, err
	}

	return replacedRecord != nil, nil
}

func (c *Collection[K, V]) storeRecord(
	tx *Txn,
	key K,
	val V,
) (replacedRecord []byte, err error) {
	if !tx.etx.Writable() {
		return nil, errReadOnly
	}
	tx.calledWriteOperation = true

	stored, err := c.codec.storedValueOf(val)
	if err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}
	replacedRecord, err = tx.etx.Records(c.name).Put(c.codec.encodedKeyOf(key), stored.encoded)
	if err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}
	if err := c.projections.recordStored(tx, key, stored, replacedRecord); err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}

	return replacedRecord, nil
}

func (c *Collection[K, V]) storeAbsentRecord(tx *Txn, key K, val V) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	stored, err := c.codec.storedValueOf(val)
	if err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}
	if err := tx.etx.Records(c.name).
		PutAbsent(c.codec.encodedKeyOf(key), stored.encoded); err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}

	if err := c.projections.recordStored(tx, key, stored, nil); err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}

	return nil
}

func (c *Collection[K, V]) PutReturning(
	tx *Txn,
	key K,
	val V,
) (replacedValue V, wasReplaced bool, err error) {
	var zero V

	replacedRecord, err := c.storeRecord(tx, key, val)
	if err != nil {
		return zero, false, err
	}
	if replacedRecord == nil {
		return zero, false, nil
	}

	replacedValue, err = c.codec.valueFrom(replacedRecord)
	if err != nil {
		return zero, false, fmt.Errorf("store %s: %w", c.name, err)
	}

	return replacedValue, true, nil
}

func (c *Collection[K, V]) Delete(tx *Txn, key K) (wasDeleted bool, err error) {
	deletedRecord, err := c.deleteRecord(tx, key)
	if err != nil {
		return false, err
	}

	return deletedRecord != nil, nil
}

func (c *Collection[K, V]) deleteRecord(tx *Txn, key K) (deletedRecord []byte, err error) {
	if !tx.etx.Writable() {
		return nil, errReadOnly
	}
	tx.calledWriteOperation = true

	deletedRecord, err = tx.etx.Records(c.name).Delete(c.codec.encodedKeyOf(key))
	if err != nil {
		return nil, fmt.Errorf("delete %s: %w", c.name, err)
	}
	if err := c.projections.recordRemoved(tx, key, deletedRecord); err != nil {
		return nil, fmt.Errorf("delete %s: %w", c.name, err)
	}

	return deletedRecord, nil
}

func (c *Collection[K, V]) DeleteReturning(
	tx *Txn,
	key K,
) (deletedValue V, wasDeleted bool, err error) {
	var zero V

	deletedRecord, err := c.deleteRecord(tx, key)
	if err != nil {
		return zero, false, err
	}
	if deletedRecord == nil {
		return zero, false, nil
	}

	deletedValue, err = c.codec.valueFrom(deletedRecord)
	if err != nil {
		return zero, false, fmt.Errorf("delete %s: %w", c.name, err)
	}

	return deletedValue, true, nil
}

func (c *Collection[K, V]) DeleteWithin(tx *Txn, keys KeyRange) (amountOfDeleted int, err error) {
	err = c.deleteRecordsWithin(tx, keys, func(deleted keyedRecord[K]) error {
		amountOfDeleted++

		return c.projections.recordRemoved(tx, deleted.key, deleted.record)
	})
	if err != nil {
		return 0, err
	}

	return amountOfDeleted, nil
}

func (c *Collection[K, V]) DeleteWithinVisiting(
	tx *Txn,
	keys KeyRange,
	visit func(K, V) error,
) error {
	return c.deleteRecordsWithin(tx, keys, func(deleted keyedRecord[K]) error {
		removed, err := c.codec.storedValueFrom(deleted.record)
		if err != nil {
			return err
		}
		if err := c.projections.valueRemoved(tx, deleted.key, removed); err != nil {
			return err
		}

		return visit(deleted.key, removed.value)
	})
}

type keyedRecord[K any] struct {
	key    K
	record []byte
}

func (c *Collection[K, V]) deleteRecordsWithin(
	tx *Txn,
	keys KeyRange,
	deleted func(keyedRecord[K]) error,
) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	storedRecords, err := c.storedRecordsWithin(tx, keys)
	if err != nil {
		return err
	}
	for _, stored := range storedRecords {
		key, err := c.deleteStored(tx, stored)
		if err != nil {
			return err
		}
		if err := deleted(keyedRecord[K]{key: key, record: stored.record}); err != nil {
			return fmt.Errorf("delete %s: %w", c.name, err)
		}
	}

	return nil
}

type storedRecord struct {
	key    []byte
	record []byte
}

func (c *Collection[K, V]) storedRecordsWithin(tx *Txn, keys KeyRange) ([]storedRecord, error) {
	var storedRecords []storedRecord
	if err := tx.etx.Records(c.name).Scan(keys, func(key, record []byte) (bool, error) {
		storedRecords = append(
			storedRecords,
			storedRecord{key: bytes.Clone(key), record: bytes.Clone(record)},
		)

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("scan %s: %w", c.name, err)
	}

	return storedRecords, nil
}

func (c *Collection[K, V]) deleteStored(tx *Txn, stored storedRecord) (K, error) {
	var zero K

	key, err := c.codec.keyFrom(stored.key)
	if err != nil {
		return zero, fmt.Errorf("delete %s: %w", c.name, err)
	}
	if err := tx.etx.Records(c.name).DeleteHeld(stored.key, stored.record); err != nil {
		return zero, fmt.Errorf("delete %s: %w", c.name, err)
	}

	return key, nil
}

func (c *Collection[K, V]) Scan(
	tx *Txn,
	keys KeyRange,
	fn func(K, V) (bool, error),
) error {
	if err := tx.etx.Records(c.name).Scan(keys, func(key, record []byte) (bool, error) {
		decodedKey, err := c.codec.keyFrom(key)
		if err != nil {
			return false, err
		}
		val, err := c.codec.valueFrom(record)
		if err != nil {
			return false, err
		}

		return fn(decodedKey, val)
	}); err != nil {
		return fmt.Errorf("scan %s: %w", c.name, err)
	}

	return nil
}

func (c *Collection[K, V]) Len(tx *Txn) (int, error) {
	return lengthOf(tx, c.name)
}

func lengthOf(tx *Txn, bucket Name) (int, error) {
	length, err := tx.etx.Records(bucket).Len()
	if err != nil {
		return 0, fmt.Errorf("length of %s: %w", bucket, err)
	}

	return length, nil
}
