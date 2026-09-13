package boltvault

import (
	"bytes"
	"fmt"

	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type boltBucket struct {
	name    vault.Name
	entries *bolt.Bucket
	lengths *bolt.Bucket
}

func (b boltBucket) Get(key []byte) ([]byte, error) {
	return b.entries.Get(key), nil
}

func (b boltBucket) Put(key []byte, val []byte) ([]byte, error) {
	previousValue := bytes.Clone(b.entries.Get(key))
	if err := b.entries.Put(key, val); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	if previousValue != nil {
		return previousValue, nil
	}

	return nil, adjustLength(b.lengths, b.name, 1)
}

func (b boltBucket) Delete(key []byte) ([]byte, error) {
	previousValue := bytes.Clone(b.entries.Get(key))
	if previousValue == nil {
		return nil, nil
	}
	if err := b.entries.Delete(key); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}
	if err := adjustLength(b.lengths, b.name, -1); err != nil {
		return nil, err
	}

	return previousValue, nil
}

func (b boltBucket) Len() (int, error) {
	return lengthOf(b.lengths, b.name)
}

func (b boltBucket) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	firstIncluded, firstExcluded := keys.Bounds()

	cursor := b.entries.Cursor()
	key, value := firstEntryFrom(cursor, firstIncluded)
	for key != nil && isBeforeFirstExcluded(key, firstExcluded) {
		keep, err := fn(key, value)
		if err != nil {
			return err
		}
		if !keep {
			return nil
		}

		key, value = cursor.Next()
	}

	return nil
}

func firstEntryFrom(cursor *bolt.Cursor, firstIncluded []byte) ([]byte, []byte) {
	if len(firstIncluded) == 0 {
		return cursor.First()
	}

	return cursor.Seek(firstIncluded)
}

func isBeforeFirstExcluded(key, firstExcluded []byte) bool {
	return firstExcluded == nil || bytes.Compare(key, firstExcluded) < 0
}
