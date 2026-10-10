// Package pebblevault is the Pebble implementation of the vault Engine. It owns the
// database directory and is the single holder of its handle; no Pebble type appears on its
// exported surface. Writes run at the same time; a write that read a key another
// write committed after it began runs again, and holds other commits once it
// conflicts too often.
package pebblevault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	bloomFilterBitsPerKey         = 10
	conflictsBeforeExclusiveWrite = 3
	amountsAtZeroKeptMessage      = "amounts at zero kept until their next change"
)

type Engine struct {
	db         *pebble.DB
	quotaBytes int64
	limits     MachineLimits
	writeSlots chan struct{}
	commitTurn chan struct{}
	history    *writeTransactionsHistory
	conflicts  WriteConflictObserver
}

type MachineLimits struct {
	BlockCacheBytes       int64
	MemtableBytes         int64
	CompactionConcurrency int
	OpenFileLimit         int
	WriteConcurrency      int
}

func OpenEngine(
	path string,
	quotaBytes int64,
	limits MachineLimits,
	stalls WriteStallObserver,
	conflicts WriteConflictObserver,
) (*Engine, error) {
	if err := os.MkdirAll(path, 0o750); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	options := optionsWithin(limits)
	options.EventListener = writeStallListenerFor(stalls)
	options.EnsureDefaults()
	imposed := machineLimitsOf(options, limits.WriteConcurrency)

	if options.Cache != nil {
		defer options.Cache.Unref()
	}

	db, err := pebble.Open(path, options)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	return &Engine{
		db:         db,
		quotaBytes: quotaBytes,
		limits:     imposed,
		writeSlots: make(chan struct{}, imposed.WriteConcurrency),
		commitTurn: make(chan struct{}, 1),
		history:    newWriteTransactionsHistory(),
		conflicts:  writeConflictObserverOrSilent(conflicts),
	}, nil
}

func optionsWithin(limits MachineLimits) *pebble.Options {
	options := &pebble.Options{
		MemTableSize: uint64(max(limits.MemtableBytes, 0)),
		MaxOpenFiles: limits.OpenFileLimit,
		Merger:       amountMerger,
	}
	addBloomFiltersAboveTheBottomLevel(options)
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

func addBloomFiltersAboveTheBottomLevel(options *pebble.Options) {
	bottomLevel := len(options.Levels) - 1
	for level := range bottomLevel {
		options.Levels[level].FilterPolicy = bloom.FilterPolicy(bloomFilterBitsPerKey)
	}
	options.Levels[bottomLevel].FilterPolicy = pebble.NoFilterPolicy
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

func machineLimitsOf(options *pebble.Options, writeConcurrency int) MachineLimits {
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
		WriteConcurrency:      max(writeConcurrency, 1),
	}
}

func (e *Engine) ProvisionRecordsBucket(_ vault.Name) error {
	return nil
}

func (e *Engine) ProvisionAmountsBucket(_ vault.Name) error {
	return nil
}

func (e *Engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.takeWriteSlot()
	defer e.freeWriteSlot()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	for range conflictsBeforeExclusiveWrite {
		conflicted, err := e.updateOptimistically(ctx, fn)
		if err != nil || !conflicted {
			return err
		}
	}

	return e.updateExclusively(ctx, fn)
}

func (e *Engine) takeWriteSlot() { e.writeSlots <- struct{}{} }

func (e *Engine) freeWriteSlot() { <-e.writeSlots }

func (e *Engine) updateOptimistically(
	ctx context.Context,
	fn func(vault.EngineTxn) error,
) (bool, error) {
	begunAfter := e.history.transactionBegan()
	defer e.history.transactionEnded(begunAfter)

	write, err := e.stage(fn)
	if err != nil {
		return false, err
	}

	e.holdCommits()
	defer e.resumeCommits()

	if key, conflicted := e.history.keyWrittenAfter(begunAfter, write.footprint); conflicted {
		reportWriteConflicted(ctx, e.conflicts, key)

		return true, release(write.changes)
	}

	return false, e.commit(ctx, write)
}

type stagedWrite struct {
	changes        *pebble.Batch
	writtenKeys    sortedKeys
	loweredAmounts *amountKeys
	footprint      *readFootprint
}

func (e *Engine) stage(fn func(vault.EngineTxn) error) (stagedWrite, error) {
	write := stagedWrite{
		changes:        e.db.NewIndexedBatch(),
		loweredAmounts: &amountKeys{},
		footprint:      newReadFootprint(),
	}
	if err := fn(pebbleTxn{
		reader:         write.changes,
		changes:        write.changes,
		loweredAmounts: write.loweredAmounts,
		footprint:      write.footprint,
	}); err != nil {
		return stagedWrite{}, errors.Join(err, release(write.changes))
	}
	writtenKeys, err := writtenKeysOf(write.changes)
	if err != nil {
		return stagedWrite{}, errors.Join(err, release(write.changes))
	}
	write.writtenKeys = writtenKeys

	return write, nil
}

func (e *Engine) holdCommits() { e.commitTurn <- struct{}{} }

func (e *Engine) resumeCommits() { <-e.commitTurn }

func (e *Engine) commit(ctx context.Context, write stagedWrite) error {
	err := errors.Join(e.commitRecorded(write), release(write.changes))
	if err != nil {
		return err
	}
	e.deleteAmountsAtZero(ctx, write.loweredAmounts)

	return nil
}

func (e *Engine) commitRecorded(write stagedWrite) error {
	if err := write.changes.Commit(pebble.Sync); err != nil {
		return commitFailureOf(err)
	}
	e.history.transactionCommitted(write.writtenKeys)

	return nil
}

func (e *Engine) deleteAmountsAtZero(ctx context.Context, loweredAmounts *amountKeys) {
	amountsAtZero, err := amountsAtZeroAmong(e.db, loweredAmounts)
	if err == nil && len(amountsAtZero) > 0 {
		err = e.commitDeletesOf(amountsAtZero)
	}
	if err != nil {
		slog.WarnContext(ctx, amountsAtZeroKeptMessage, slog.Any("error", err))
	}
}

func amountsAtZeroAmong(
	committedState pebble.Reader,
	loweredAmounts *amountKeys,
) ([][]byte, error) {
	amountEntries := storedEntries{reader: committedState}
	var amountsAtZero [][]byte
	for _, key := range loweredAmounts.keys {
		raw, err := amountEntries.valueAt(key)
		if err != nil {
			return nil, err
		}
		if raw == nil {
			continue
		}
		amount, err := countFrom(raw)
		if err != nil {
			return nil, err
		}
		if amount == 0 {
			amountsAtZero = append(amountsAtZero, key)
		}
	}

	return amountsAtZero, nil
}

func (e *Engine) commitDeletesOf(keys [][]byte) error {
	deletes := e.db.NewBatch()
	for _, key := range keys {
		if err := deletes.Delete(key, pebble.NoSync); err != nil {
			return errors.Join(fmt.Errorf("delete amount at zero: %w", err), release(deletes))
		}
	}

	return errors.Join(deletes.Commit(pebble.NoSync), release(deletes))
}

func (e *Engine) updateExclusively(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.holdCommits()
	defer e.resumeCommits()

	begunAfter := e.history.transactionBegan()
	defer e.history.transactionEnded(begunAfter)

	reportExclusiveWrite(ctx, e.conflicts)
	write, err := e.stage(fn)
	if err != nil {
		return err
	}

	return e.commit(ctx, write)
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

	snapshot := e.db.NewSnapshot()

	if err := fn(pebbleTxn{reader: snapshot, loweredAmounts: &amountKeys{}}); err != nil {
		return errors.Join(fmt.Errorf("read storage: %w", err), release(snapshot))
	}

	return release(snapshot)
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

	heldBytes, err := heldBytesOf(e.db)
	if err != nil {
		return 0, err
	}
	estimatedAmountsBytes, err := e.db.EstimateDiskUsage(amountsRegion.boundsFor(vault.EveryKey()))
	if err != nil {
		return 0, fmt.Errorf("estimate amount bytes: %w", err)
	}

	return heldBytes + signed(estimatedAmountsBytes), nil
}
