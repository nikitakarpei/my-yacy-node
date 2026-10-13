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
	collection, err := v.RegisterCollection(bucket, keys, values)
	if err != nil {
		return nil, err
	}
	if err := followEach(projections.buckets()); err != nil {
		return nil, err
	}
	collection.projections = projections
	if err := v.rebuildStaleProjections(ctx, collection, projections); err != nil {
		return nil, err
	}

	return collection, nil
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

	record, err := tx.etx.Records(c.name).Get(c.keys.Encode(key).encoded)
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

	payload, err := c.values.Encode(val)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", c.name, err)
	}

	record := recordFrom(payload)
	replacedRecord, err = tx.etx.Records(c.name).Put(c.keys.Encode(key).encoded, record)
	if err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}
	storedValue := storedForm[V]{form: val, stored: record}
	if err := c.followStored(tx, key, storedValue, replacedRecord); err != nil {
		return nil, err
	}

	return replacedRecord, nil
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

	deletedRecord, err = tx.etx.Records(c.name).Delete(c.keys.Encode(key).encoded)
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
	firstKey []byte,
	limit int,
	projections Projections[K, V],
) (nextKey []byte, err error) {
	projected := 0
	if err := tx.etx.Records(c.name).Scan(
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
