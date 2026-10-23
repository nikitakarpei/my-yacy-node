package vault

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const removalPassesPerLifetime = 32

var errLifetimeTooShort = errors.New("expiring collection lifetime is under one second")

type ExpiringCollection[K, V any] struct {
	name        Name
	codec       recordCodec[K, V]
	projections attachedProjections[K, V]
	lifetime    time.Duration
}

//nolint:revive // an expiring collection names its bucket, layout, codec, lifetime and projections
func (v *Vault) RegisterExpiringCollection[K, V any](
	ctx context.Context,
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
	lifetime time.Duration,
	projections Projections[K, V],
) (*ExpiringCollection[K, V], error) {
	if lifetime < time.Second {
		return nil, fmt.Errorf("%w: %s", errLifetimeTooShort, bucket)
	}
	if err := v.provisionExpiringRecordsBucket(bucket); err != nil {
		return nil, err
	}
	codec := recordCodec[K, V]{keys: keys, values: values}
	expiring := &ExpiringCollection[K, V]{
		name:        bucket,
		codec:       codec,
		projections: attachedProjections[K, V]{projections: projections, codec: codec},
		lifetime:    lifetime,
	}
	if err := v.attachProjections(
		ctx,
		bucket,
		expiring.projections,
		func(tx *Txn) storedRecords { return tx.etx.ExpiringRecords(bucket) },
	); err != nil {
		return nil, err
	}
	v.removals.start(ctx, expiryRemoval{
		storage:  v,
		clock:    v.clock,
		bucket:   bucket,
		records:  expiring,
		interval: lifetime / removalPassesPerLifetime,
		observer: v.expiryObserver,
	})

	return expiring, nil
}

func (c *ExpiringCollection[K, V]) Get(tx *Txn, key K) (V, bool, error) {
	var zero V

	record, err := tx.etx.ExpiringRecords(c.name).Get(c.codec.encodedKeyOf(key), tx.at)
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

func (c *ExpiringCollection[K, V]) Put(tx *Txn, key K, val V) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	stored, err := c.codec.storedValueOf(val)
	if err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}
	replacedRecord, err := tx.etx.ExpiringRecords(c.name).Put(
		c.codec.encodedKeyOf(key),
		stored.encoded,
		tx.at.Add(c.lifetime),
	)
	if err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}
	if err := c.projections.recordStored(tx, key, stored, replacedRecord); err != nil {
		return fmt.Errorf("store %s: %w", c.name, err)
	}

	return nil
}

func (c *ExpiringCollection[K, V]) expiredKeysFrom(
	tx *Txn,
	resume []byte,
) (expiredKeys [][]byte, nextResume []byte, err error) {
	expiredKeys, nextResume, err = tx.etx.ExpiringRecords(c.name).ExpiredKeys(
		resume,
		tx.at,
		expiredRecordsPerChunk,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list expired %s: %w", c.name, err)
	}

	return expiredKeys, nextResume, nil
}

func (c *ExpiringCollection[K, V]) forgetExpired(
	tx *Txn,
	expiredKeys [][]byte,
) (amountForgotten int, err error) {
	if !tx.etx.Writable() {
		return 0, errReadOnly
	}
	tx.calledWriteOperation = true

	if err := tx.etx.ExpiringRecords(c.name).ForgetExpired(
		expiredKeys,
		tx.at,
		func(key, forgottenRecord []byte) error {
			amountForgotten++

			return c.recordForgotten(tx, key, forgottenRecord)
		},
	); err != nil {
		return 0, fmt.Errorf("forget expired %s: %w", c.name, err)
	}

	return amountForgotten, nil
}

func (c *ExpiringCollection[K, V]) recordForgotten(tx *Txn, key, forgottenRecord []byte) error {
	decodedKey, err := c.codec.keyFrom(key)
	if err == nil {
		err = c.projections.recordRemoved(tx, decodedKey, forgottenRecord)
	}
	if errors.Is(err, errBadRecord) {
		return undecodableRecordError{key: key, cause: err}
	}

	return err
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
