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
	keys        KeyLayout[K]
	values      ValueCodec[V]
	projections Projections[K, V]
}

func (v *Vault) RegisterCollection[K, V any](
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
) (*Collection[K, V], error) {
	if err := v.provisionRecordsBucket(bucket); err != nil {
		return nil, err
	}

	return &Collection[K, V]{name: bucket, keys: keys, values: values}, nil
}

func (v *Vault) RegisterProjectedCollection[K, V any](
	ctx context.Context,
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
	projections Projections[K, V],
) (*Collection[K, V], error) {
	return v.registerProjected(ctx, &Collection[K, V]{
		name:        bucket,
		keys:        keys,
		values:      values,
		projections: projections,
	})
}

func (v *Vault) registerProjected[K, V any](
	ctx context.Context,
	collection *Collection[K, V],
) (*Collection[K, V], error) {
	if err := v.provisionRecordsBucket(collection.name); err != nil {
		return nil, err
	}
	if err := v.followProjections(ctx, collection, collection.storedRecordsIn); err != nil {
		return nil, err
	}

	return collection, nil
}

func (v *Vault) followProjections[K, V any](
	ctx context.Context,
	collection *Collection[K, V],
	storedRecordsIn func(*Txn) storedRecords,
) error {
	if collection.projections == nil {
		return nil
	}
	if err := followEach(collection.projections.buckets()); err != nil {
		return err
	}

	return v.rebuildStaleProjections(ctx, collection, storedRecordsIn)
}

type storedRecords interface {
	Scan(keys KeyRange, fn func(key, record []byte) (bool, error)) error
}

func (c *Collection[K, V]) storedRecordsIn(tx *Txn) storedRecords {
	return tx.etx.Records(c.name)
}

func followEach(buckets []projectionBucket) error {
	for _, bucket := range buckets {
		if err := bucket.follow(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection[K, V]) Get(tx *Txn, key K) (V, bool, error) {
	var zero V

	record, err := tx.etx.Records(c.name).Get(c.encodedKeyOf(key))
	if err != nil {
		return zero, false, fmt.Errorf("read %s: %w", c.name, err)
	}
	if record == nil {
		return zero, false, nil
	}

	val, err := c.valueFrom(record)
	if err != nil {
		return zero, false, err
	}

	return val, true, nil
}

func (c *Collection[K, V]) encodedKeyOf(key K) []byte {
	return c.keys.Encode(key).encoded
}

func (c *Collection[K, V]) valueFrom(record []byte) (V, error) {
	var zero V

	payload, err := payloadOf(record)
	if err != nil {
		return zero, fmt.Errorf("record in %s: %w", c.name, err)
	}

	val, err := c.values.Decode(payload)
	if err != nil {
		return zero, fmt.Errorf("decode %s: %w", c.name, err)
	}

	return val, nil
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

	record, err := c.recordOf(val)
	if err != nil {
		return nil, err
	}
	replacedRecord, err = tx.etx.Records(c.name).Put(c.encodedKeyOf(key), record)
	if err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}
	storedValue := storedForm[V]{form: val, stored: record}
	if err := c.followStored(tx, key, storedValue, replacedRecord); err != nil {
		return nil, err
	}

	return replacedRecord, nil
}

func (c *Collection[K, V]) recordOf(val V) ([]byte, error) {
	payload, err := c.values.Encode(val)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", c.name, err)
	}

	return recordFrom(payload), nil
}

func (c *Collection[K, V]) followStored(
	tx *Txn,
	key K,
	storedValue storedForm[V],
	replacedRecord []byte,
) error {
	if c.projections == nil {
		return nil
	}
	if replacedRecord == nil {
		return c.projections.recordAdded(tx, key, storedValue)
	}
	replacedValue, err := c.storedFormFrom(replacedRecord)
	if err != nil {
		return err
	}

	return c.projections.recordReplaced(tx, key, replacedValue, storedValue)
}

func (c *Collection[K, V]) storedFormFrom(record []byte) (storedForm[V], error) {
	val, err := c.valueFrom(record)
	if err != nil {
		return storedForm[V]{}, err
	}

	return storedForm[V]{form: val, stored: record}, nil
}

func (c *Collection[K, V]) storeAbsentRecord(tx *Txn, key K, val V) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	record, err := c.recordOf(val)
	if err != nil {
		return err
	}
	if err := tx.etx.Records(c.name).PutAbsent(c.encodedKeyOf(key), record); err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}

	return c.followStored(tx, key, storedForm[V]{form: val, stored: record}, nil)
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

	replacedValue, err = c.valueFrom(replacedRecord)
	if err != nil {
		return zero, false, err
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

	deletedRecord, err = tx.etx.Records(c.name).Delete(c.encodedKeyOf(key))
	if err != nil {
		return nil, fmt.Errorf("delete %s: %w", c.name, err)
	}
	if err := c.followDeleted(tx, key, deletedRecord); err != nil {
		return nil, err
	}

	return deletedRecord, nil
}

func (c *Collection[K, V]) followDeleted(tx *Txn, key K, deletedRecord []byte) error {
	if c.projections == nil || deletedRecord == nil {
		return nil
	}
	deletedValue, err := c.storedFormFrom(deletedRecord)
	if err != nil {
		return err
	}

	return c.projections.recordRemoved(tx, key, deletedValue)
}

func (c *Collection[K, V]) followForgotten(tx *Txn, key, forgottenRecord []byte) error {
	if c.projections == nil {
		return nil
	}
	decodedKey, err := c.keys.Decode(key)
	if err != nil {
		return undecodableRecordError{key: key, cause: fmt.Errorf("decode %s key: %w", c.name, err)}
	}
	forgottenValue, err := c.storedFormFrom(forgottenRecord)
	if err != nil {
		return undecodableRecordError{key: key, cause: err}
	}

	return c.projections.recordRemoved(tx, decodedKey, forgottenValue)
}

type undecodableRecordError struct {
	cause error
	key   []byte
}

func (e undecodableRecordError) Error() string {
	return e.cause.Error()
}

func (e undecodableRecordError) Unwrap() error {
	return e.cause
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

	deletedValue, err = c.valueFrom(deletedRecord)
	if err != nil {
		return zero, false, err
	}

	return deletedValue, true, nil
}

func (c *Collection[K, V]) DeleteWithin(tx *Txn, keys KeyRange) (amountOfDeleted int, err error) {
	if !tx.etx.Writable() {
		return 0, errReadOnly
	}
	tx.calledWriteOperation = true

	storedRecords, err := c.storedRecordsWithin(tx, keys)
	if err != nil {
		return 0, err
	}
	for _, stored := range storedRecords {
		if err := c.deleteStored(tx, stored); err != nil {
			return 0, err
		}
	}

	return len(storedRecords), nil
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

func (c *Collection[K, V]) deleteStored(tx *Txn, stored storedRecord) error {
	key, err := c.keys.Decode(stored.key)
	if err != nil {
		return fmt.Errorf("decode %s key: %w", c.name, err)
	}
	if err := tx.etx.Records(c.name).DeleteHeld(stored.key, stored.record); err != nil {
		return fmt.Errorf("delete %s: %w", c.name, err)
	}

	return c.followDeleted(tx, key, stored.record)
}

func (c *Collection[K, V]) Scan(
	tx *Txn,
	keys KeyRange,
	fn func(K, V) (bool, error),
) error {
	if err := tx.etx.Records(c.name).Scan(keys, func(key, record []byte) (bool, error) {
		decodedKey, err := c.keys.Decode(key)
		if err != nil {
			return false, fmt.Errorf("decode %s key: %w", c.name, err)
		}
		val, err := c.valueFrom(record)
		if err != nil {
			return false, err
		}

		return fn(decodedKey, val)
	}); err != nil {
		return fmt.Errorf("scan %s: %w", c.name, err)
	}

	return nil
}

func (c *Collection[K, V]) projectRecordsFrom(
	tx *Txn,
	records storedRecords,
	firstKey []byte,
	limit int,
	projections Projections[K, V],
) (nextKey []byte, err error) {
	projected := 0
	if err := records.Scan(
		KeyRange{firstIncluded: firstKey},
		func(key, record []byte) (bool, error) {
			if projected == limit {
				nextKey = bytes.Clone(key)

				return false, nil
			}
			projected++

			return true, c.projectRecord(tx, key, record, projections)
		},
	); err != nil {
		return nil, fmt.Errorf("project %s: %w", c.name, err)
	}

	return nextKey, nil
}

func (c *Collection[K, V]) projectRecord(
	tx *Txn,
	key, record []byte,
	projections Projections[K, V],
) error {
	decodedKey, err := c.keys.Decode(key)
	if err != nil {
		return fmt.Errorf("decode %s key: %w", c.name, err)
	}
	storedValue, err := c.storedFormFrom(record)
	if err != nil {
		return err
	}

	return projections.recordAdded(tx, decodedKey, storedValue)
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
