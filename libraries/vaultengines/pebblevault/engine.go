// Package pebblevault is the Pebble implementation of the vault Engine. It owns
// the log-structured database directory and is the single holder of the database
// handle; no caller receives the raw handle and no Pebble type appears on its
// exported surface.
package pebblevault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type engine struct {
	db         *pebble.DB
	quotaBytes int64
	writing    sync.Mutex
}

func Open(
	path string,
	quotaBytes int64,
	cacheBytes int64,
	observer vault.TransactionObserver,
) (*vault.Vault, error) {
	opened, err := OpenEngine(path, quotaBytes, cacheBytes)
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

func OpenEngine(path string, quotaBytes int64, cacheBytes int64) (vault.Engine, error) {
	if err := os.MkdirAll(path, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	cache := pebble.NewCache(cacheBytes)
	defer cache.Unref()

	db, err := pebble.Open(path, &pebble.Options{Cache: cache})
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	return &engine{db: db, quotaBytes: quotaBytes}, nil
}

func (e *engine) Provision(_ vault.Name) error {
	return nil
}

func (e *engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.writing.Lock()
	defer e.writing.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	staged := e.db.NewIndexedBatch()

	if err := fn(pebbleTxn{reader: staged, staged: staged}); err != nil {
		return errors.Join(err, release(staged))
	}

	return errors.Join(commitFailureOf(staged.Commit(pebble.Sync)), release(staged))
}

func commitFailureOf(err error) error {
	if err == nil {
		return nil
	}
	if cause, atCapacity := capacityCauseOf(err); atCapacity {
		return capacityError{cause: cause, err: vault.ErrAtCapacity}
	}

	return fmt.Errorf("update storage: %w", err)
}

func release(handle io.Closer) error {
	if err := handle.Close(); err != nil {
		return fmt.Errorf("release storage handle: %w", err)
	}

	return nil
}

func (e *engine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	committed := e.db.NewSnapshot()

	if err := fn(pebbleTxn{reader: committed}); err != nil {
		return errors.Join(fmt.Errorf("read storage: %w", err), release(committed))
	}

	return release(committed)
}

func (e *engine) Close() error {
	if err := e.db.Close(); err != nil {
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

	return heldBytesOf(bucketTalliesIn(e.db))
}
