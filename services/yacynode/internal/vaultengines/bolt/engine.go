// Package bolt is the bbolt implementation of the vault Engine. It owns the
// embedded database file and is the single holder of the database handle; no
// caller receives the raw handle and no bolt type appears on its exported
// surface.
package bolt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

type engine struct {
	db         *bolt.DB
	quotaBytes int64
}

func Open(
	path string,
	quotaBytes int64,
	observer vault.TransactionObserver,
) (*vault.Vault, error) {
	opened, err := OpenEngine(path, quotaBytes)
	if err != nil {
		return nil, err
	}

	vaulted, err := vault.New(opened, observer)
	if err != nil {
		if closeErr := opened.Close(); closeErr != nil {
			return nil, fmt.Errorf("initialize storage: %w: %w", err, closeErr)
		}

		return nil, fmt.Errorf("initialize storage: %w", err)
	}

	return vaulted, nil
}

func OpenEngine(path string, quotaBytes int64) (vault.Engine, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	return &engine{db: db, quotaBytes: quotaBytes}, nil
}

func (e *engine) Provision(name vault.Name) error {
	if err := e.db.Update(func(tx *bolt.Tx) error {
		if _, createErr := tx.CreateBucketIfNotExists([]byte(name)); createErr != nil {
			return fmt.Errorf("create bucket: %w", createErr)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("provision bucket %s: %w", name, err)
	}

	return nil
}

func (e *engine) Update(_ context.Context, fn func(vault.EngineTxn) error) error {
	if err := e.db.Update(func(tx *bolt.Tx) error {
		return fn(boltTxn{tx: tx, writable: true})
	}); err != nil {
		if cause, atCapacity := capacityCauseOf(err); atCapacity {
			return capacityError{cause: cause, err: vault.ErrAtCapacity}
		}

		return fmt.Errorf("update storage: %w", err)
	}

	return nil
}

func (e *engine) View(_ context.Context, fn func(vault.EngineTxn) error) error {
	if err := e.db.View(func(tx *bolt.Tx) error {
		return fn(boltTxn{tx: tx, writable: false})
	}); err != nil {
		return fmt.Errorf("read storage: %w", err)
	}

	return nil
}

func (e *engine) Close() error {
	err := e.db.Close()
	if err != nil {
		return fmt.Errorf("close storage: %w", err)
	}

	return nil
}

type boltTxn struct {
	tx       *bolt.Tx
	writable bool
}

func (t boltTxn) Writable() bool { return t.writable }

func (t boltTxn) Bucket(name vault.Name) vault.EngineBucket {
	return boltBucket{bucket: t.tx.Bucket([]byte(name))}
}

type boltBucket struct {
	bucket *bolt.Bucket
}

func (b boltBucket) Get(key []byte) []byte {
	return b.bucket.Get(key)
}

func (b boltBucket) Put(key []byte, val []byte) error {
	if err := b.bucket.Put(key, val); err != nil {
		return fmt.Errorf("store: %w", err)
	}

	return nil
}

func (b boltBucket) Delete(key []byte) error {
	if err := b.bucket.Delete(key); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (b boltBucket) Scan(keys vaultkey.KeyRange, fn func(key, value []byte) (bool, error)) error {
	firstIncluded, firstExcluded := keys.Bounds()

	cursor := b.bucket.Cursor()
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
