package vault

import (
	"errors"
	"time"
)

type RetentionIndex[R, K any] struct {
	entries *ValueIndex[R, retainedKey[K]]
}

type retainedKey[K any] struct {
	retainedSince time.Time
	key           K
}

func (v *Vault) RegisterRetentionIndex[R, K any](
	bucket Name,
	version string,
	keys KeyLayout[K],
	retainedSinceOf func(R) time.Time,
	keyOf func(R) K,
) (*RetentionIndex[R, K], error) {
	entries, err := v.RegisterValueIndex(
		bucket,
		version,
		retainedKeyLayoutFrom(keys),
		func(record R) retainedKey[K] {
			return retainedKey[K]{retainedSince: retainedSinceOf(record), key: keyOf(record)}
		},
	)
	if err != nil {
		return nil, err
	}

	return &RetentionIndex[R, K]{entries: entries}, nil
}

func retainedKeyLayoutFrom[K any](keys KeyLayout[K]) KeyLayout[retainedKey[K]] {
	return KeyLayout[retainedKey[K]]{
		encode: func(retained retainedKey[K]) Key {
			encoded := TimeInSecondsKeyPart.appendTo(
				make([]byte, 0, keyCapacity),
				retained.retainedSince,
			)

			return Key{encoded: append(encoded, keys.Encode(retained.key).encoded...)}
		},
		decode: func(storedKey []byte) (retainedKey[K], error) {
			retainedSince, rest, sinceErr := TimeInSecondsKeyPart.parseFrom(
				orderedBytes{encoded: storedKey},
			)
			key, keyErr := keys.Decode(rest.encoded)

			retained := retainedKey[K]{retainedSince: retainedSince, key: key}

			return retained, errors.Join(sinceErr, keyErr)
		},
	}
}

func (i *RetentionIndex[R, K]) KeysBefore(tx *Txn, cutoff time.Time, limit int) ([]K, error) {
	keys := make([]K, 0, limit)
	err := i.entries.Scan(
		tx,
		TimeInSecondsKeyPart.keysBefore(cutoff),
		func(retained retainedKey[K]) (bool, error) {
			if len(keys) == limit {
				return false, nil
			}
			keys = append(keys, retained.key)

			return true, nil
		},
	)

	return keys, err
}

func (i *RetentionIndex[R, K]) bucketName() Name {
	return i.entries.bucketName()
}

func (i *RetentionIndex[R, K]) follow() error {
	return i.entries.follow()
}

func (i *RetentionIndex[R, K]) registeredVersion() string {
	return i.entries.registeredVersion()
}

func (i *RetentionIndex[R, K]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	return i.entries.clearFrom(tx, firstKey, limit)
}

func (i *RetentionIndex[R, K]) recordAdded(tx *Txn, record storedForm[R]) error {
	return i.entries.recordAdded(tx, record)
}

func (i *RetentionIndex[R, K]) recordRemoved(tx *Txn, record storedForm[R]) error {
	return i.entries.recordRemoved(tx, record)
}

func (i *RetentionIndex[R, K]) recordReplaced(tx *Txn, previous, current storedForm[R]) error {
	return i.entries.recordReplaced(tx, previous, current)
}
