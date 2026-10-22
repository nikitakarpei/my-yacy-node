package vault

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const removalPassesPerLifetime = 32

var (
	errLifetimeTooShort     = errors.New("expiring collection lifetime is under one second")
	errExpiryObserverAbsent = errors.New("expiring collection needs an expiry observer")
)

type ExpiringCollection[K, V any] struct {
	collection *Collection[K, V]
	lifetime   time.Duration
}

//nolint:revive // an expiring collection names its bucket, layout, codec, lifetime, observer and projections
func (v *Vault) RegisterExpiringCollection[K, V any](
	ctx context.Context,
	bucket Name,
	keys KeyLayout[K],
	values ValueCodec[V],
	lifetime time.Duration,
	observer ExpiryObserver,
	projections Projections[K, V],
) (*ExpiringCollection[K, V], error) {
	if lifetime < time.Second {
		return nil, fmt.Errorf("%w: %s", errLifetimeTooShort, bucket)
	}
	if observer == nil {
		return nil, fmt.Errorf("%w: %s", errExpiryObserverAbsent, bucket)
	}
	if err := v.provisionExpiringRecordsBucket(bucket); err != nil {
		return nil, err
	}
	expiring := &ExpiringCollection[K, V]{
		collection: &Collection[K, V]{
			name:        bucket,
			keys:        keys,
			values:      values,
			projections: projections,
		},
		lifetime: lifetime,
	}
	if err := v.followProjections(ctx, expiring.collection, expiring.storedRecordsIn); err != nil {
		return nil, err
	}
	v.removals.start(ctx, expiryRemoval{
		storage:  v,
		clock:    v.clock,
		bucket:   bucket,
		records:  expiring,
		interval: lifetime / removalPassesPerLifetime,
		observer: observer,
	})

	return expiring, nil
}

func (c *ExpiringCollection[K, V]) storedRecordsIn(tx *Txn) storedRecords {
	return c.expiringRecordsIn(tx)
}

func (c *ExpiringCollection[K, V]) expiringRecordsIn(tx *Txn) EngineExpiringRecords {
	return tx.etx.ExpiringRecords(c.collection.name)
}

func (c *ExpiringCollection[K, V]) Get(tx *Txn, key K) (V, bool, error) {
	var zero V

	record, err := c.expiringRecordsIn(tx).Get(c.collection.encodedKeyOf(key), tx.at)
	if err != nil {
		return zero, false, fmt.Errorf("read %s: %w", c.collection.name, err)
	}
	if record == nil {
		return zero, false, nil
	}

	val, err := c.collection.valueFrom(record)
	if err != nil {
		return zero, false, err
	}

	return val, true, nil
}

func (c *ExpiringCollection[K, V]) Put(tx *Txn, key K, val V) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	record, err := c.collection.recordOf(val)
	if err != nil {
		return err
	}
	replacedRecord, err := c.expiringRecordsIn(tx).Put(
		c.collection.encodedKeyOf(key),
		record,
		tx.at.Add(c.lifetime),
	)
	if err != nil {
		return fmt.Errorf("store %s: %w", c.collection.name, err)
	}

	return c.collection.followStored(
		tx,
		key,
		storedForm[V]{form: val, stored: record},
		replacedRecord,
	)
}

func (c *ExpiringCollection[K, V]) expiredKeysFrom(
	tx *Txn,
	resume []byte,
) (expiredKeys [][]byte, nextResume []byte, err error) {
	expiredKeys, nextResume, err = c.expiringRecordsIn(tx).ExpiredKeys(
		resume,
		tx.at,
		expiredRecordsPerChunk,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list expired %s: %w", c.collection.name, err)
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

	if err := c.expiringRecordsIn(tx).ForgetExpired(
		expiredKeys,
		tx.at,
		func(key, forgottenRecord []byte) error {
			amountForgotten++

			return c.collection.followForgotten(tx, key, forgottenRecord)
		},
	); err != nil {
		return 0, fmt.Errorf("forget expired %s: %w", c.collection.name, err)
	}

	return amountForgotten, nil
}
