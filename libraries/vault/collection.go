package vault

import (
	"errors"
	"fmt"
)

var errReadOnly = errors.New("write inside read-only transaction")

type Collection[K, V any] struct {
	name   Name
	keys   KeyLayout[K]
	values ValueCodec[V]
}

func (v *Vault) RegisterCollection[K, V any](
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
) (*Collection[K, V], error) {
	if err := v.provision(bucket); err != nil {
		return nil, err
	}

	return &Collection[K, V]{name: bucket, keys: keys, values: values}, nil
}

func (c *Collection[K, V]) Get(tx *Txn, key K) (V, bool, error) {
	var zero V

	record, err := tx.etx.Bucket(c.name).Get(c.keys.Encode(key).Bytes())
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

func (c *Collection[K, V]) Put(tx *Txn, key K, val V) error {
	_, err := c.storeRecord(tx, key, val)

	return err
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

	replacedRecord, err = tx.etx.Bucket(c.name).Put(
		c.keys.Encode(key).Bytes(),
		recordFrom(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("store %s: %w", c.name, err)
	}

	return replacedRecord, nil
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

	deletedRecord, err = tx.etx.Bucket(c.name).Delete(c.keys.Encode(key).Bytes())
	if err != nil {
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
	if err := tx.etx.Bucket(c.name).Scan(keys, func(key, record []byte) (bool, error) {
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

func (c *Collection[K, V]) Len(tx *Txn) (int, error) {
	return lengthOf(tx, c.name)
}

func lengthOf(tx *Txn, bucket Name) (int, error) {
	length, err := tx.etx.Bucket(bucket).Len()
	if err != nil {
		return 0, fmt.Errorf("length of %s: %w", bucket, err)
	}

	return length, nil
}
