// Package boltvault is the bbolt implementation of the vault Engine. It owns the
// embedded database file and is the single holder of the database handle; no
// caller receives the raw handle and no bolt type appears on its exported
// surface.
package boltvault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type engine struct {
	db         *bolt.DB
	quotaBytes int64
}

// WriteBatch is how many write transactions the database merges into one commit
// and how long it waits to fill a merge. A zero field keeps the bbolt default.
type WriteBatch struct {
	MaximumWrites int
	MaximumDelay  time.Duration
}

func Open(
	path string,
	quotaBytes int64,
	writeBatch WriteBatch,
	observer vault.TransactionObserver,
) (*vault.Vault, error) {
	opened, err := OpenEngine(path, quotaBytes, writeBatch)
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

func OpenEngine(path string, quotaBytes int64, writeBatch WriteBatch) (vault.Engine, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	db, err := bolt.Open(path, 0o600, &bolt.Options{FreelistType: bolt.FreelistMapType})
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}
	applyWriteBatch(db, writeBatch)

	opened := &engine{db: db, quotaBytes: quotaBytes}
	if err := opened.createBucket(lengthBucket); err != nil {
		return nil, errors.Join(err, release(db))
	}

	return opened, nil
}

func applyWriteBatch(db *bolt.DB, writeBatch WriteBatch) {
	if writeBatch.MaximumWrites > 0 {
		db.MaxBatchSize = writeBatch.MaximumWrites
	}
	if writeBatch.MaximumDelay > 0 {
		db.MaxBatchDelay = writeBatch.MaximumDelay
	}
}

func release(handle io.Closer) error {
	if err := handle.Close(); err != nil {
		return fmt.Errorf("release storage handle: %w", err)
	}

	return nil
}

var errReservedBucket = errors.New("bucket name reserved for storage internals")

func (e *engine) Provision(name vault.Name) error {
	if name == lengthBucket {
		return fmt.Errorf("provision bucket %s: %w", name, errReservedBucket)
	}

	return e.createBucket(name)
}

func (e *engine) createBucket(name vault.Name) error {
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

func (e *engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	if err := e.db.Batch(func(tx *bolt.Tx) error {
		return fn(boltTxn{tx: tx, writable: true})
	}); err != nil {
		if cause, atCapacity := capacityCauseOf(err); atCapacity {
			return capacityError{cause: cause, err: vault.ErrAtCapacity}
		}

		return fmt.Errorf("update storage: %w", err)
	}

	return nil
}

func (e *engine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

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

func (e *engine) QuotaBytes() int64 {
	return e.quotaBytes
}

func (e *engine) UsedBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	var used int64
	if err := e.db.View(func(tx *bolt.Tx) error {
		stats := e.db.Stats()
		pageSize := int64(e.db.Info().PageSize)
		free := int64(stats.FreePageN+stats.PendingPageN) * pageSize
		used = tx.Size() - free

		return nil
	}); err != nil {
		return 0, fmt.Errorf("read storage stats: %w", err)
	}
	if used < 0 {
		used = 0
	}

	return used, nil
}
