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

type Engine struct {
	db         *pebble.DB
	quotaBytes int64
	limits     MachineLimits
	writing    sync.Mutex
}

type MachineLimits struct {
	BlockCacheBytes       int64
	MemtableBytes         int64
	CompactionConcurrency int
	OpenFileLimit         int
}

func OpenEngine(
	path string,
	quotaBytes int64,
	limits MachineLimits,
	stalls WriteStallObserver,
) (*Engine, error) {
	if err := os.MkdirAll(path, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	options := optionsWithin(limits)
	options.EventListener = writeStallListenerFor(stalls)
	options.EnsureDefaults()
	imposed := machineLimitsOf(options)

	if options.Cache != nil {
		defer options.Cache.Unref()
	}

	db, err := pebble.Open(path, options)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	return &Engine{db: db, quotaBytes: quotaBytes, limits: imposed}, nil
}

func optionsWithin(limits MachineLimits) *pebble.Options {
	options := &pebble.Options{
		MemTableSize: uint64(max(limits.MemtableBytes, 0)),
		MaxOpenFiles: limits.OpenFileLimit,
	}
	if limits.BlockCacheBytes > 0 {
		options.Cache = pebble.NewCache(limits.BlockCacheBytes)
	}
	if limits.CompactionConcurrency > 0 {
		options.CompactionConcurrencyRange = func() (int, int) {
			return 1, limits.CompactionConcurrency
		}
	}

	return options
}

func writeStallListenerFor(observer WriteStallObserver) *pebble.EventListener {
	if observer == nil {
		observer = silentWriteStallObserver{}
	}

	return &pebble.EventListener{
		WriteStallBegin: func(info pebble.WriteStallBeginInfo) {
			reportWriteStallBegan(observer, writeStallCauseOf(info.Reason))
		},
		WriteStallEnd: func() { reportWriteStallEnded(observer) },
	}
}

func machineLimitsOf(options *pebble.Options) MachineLimits {
	blockCacheBytes := options.CacheSize
	if options.Cache != nil {
		blockCacheBytes = options.Cache.MaxSize()
	}
	_, compactionConcurrency := options.CompactionConcurrencyRange()

	return MachineLimits{
		BlockCacheBytes:       blockCacheBytes,
		MemtableBytes:         signed(options.MemTableSize),
		CompactionConcurrency: compactionConcurrency,
		OpenFileLimit:         options.MaxOpenFiles,
	}
}

func (e *Engine) Provision(_ vault.Name) error {
	return nil
}

func (e *Engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
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

func (e *Engine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	committed := e.db.NewSnapshot()

	if err := fn(pebbleTxn{reader: committed}); err != nil {
		return errors.Join(fmt.Errorf("read storage: %w", err), release(committed))
	}

	return release(committed)
}

func (e *Engine) Close() error {
	if err := e.db.Close(); err != nil {
		return fmt.Errorf("close storage: %w", err)
	}

	return nil
}

func (e *Engine) QuotaBytes() int64 {
	return e.quotaBytes
}

func (e *Engine) Condition() EngineCondition {
	return engineConditionOf(e.db.Metrics(), e.limits)
}

func (e *Engine) UsedBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	return heldBytesOf(bucketTalliesIn(e.db))
}
