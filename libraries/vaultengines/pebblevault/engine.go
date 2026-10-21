// Package pebblevault is the Pebble implementation of the vault Engine. It owns the
// database directory and is the single holder of its handle; no Pebble type appears on its
// exported surface. Writes run at the same time; a write that read a key another
// write committed after it began runs again, and holds other commits once it
// conflicts too often. Commits share disk syncs; a write returns once its commit
// is on disk.
package pebblevault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/compressionratios"
)

const (
	bloomFilterBitsPerKey         = 10
	blockBytes                    = 16 << 10
	conflictsBeforeExclusiveWrite = 3
)

type Engine struct {
	db         *pebble.DB
	quotaBytes int64
	limits     MachineLimits
	writeSlots chan struct{}
	commitTurn chan struct{}
	history    *writeTransactionsHistory
	conflicts  WriteConflictObserver
	scans      *scanTally
	surveyor   *compressionSurveyor
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

	surveyor := newCompressionSurveyor()
	options := optionsWithin(limits)
	options.EventListener = eventListenerFor(stalls)
	options.EnsureDefaults()
	options.AddEventListener(surveyor.eventListener())
	imposed := machineLimitsOf(options, limits.WriteConcurrency)

	if options.Cache != nil {
		defer options.Cache.Unref()
	}

	db, err := pebble.Open(path, options)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}
	surveyor.start(db)

	return &Engine{
		db:         db,
		quotaBytes: quotaBytes,
		limits:     imposed,
		writeSlots: make(chan struct{}, imposed.WriteConcurrency),
		commitTurn: make(chan struct{}, 1),
		history:    newWriteTransactionsHistory(),
		conflicts:  writeConflictObserverOrSilent(conflicts),
		scans:      &scanTally{},
		surveyor:   surveyor,
	}, nil
}

func optionsWithin(limits MachineLimits) *pebble.Options {
	options := &pebble.Options{
		MemTableSize: uint64(max(limits.MemtableBytes, 0)),
		MaxOpenFiles: limits.OpenFileLimit,
		Merger:       regionMerger,
	}
	addBloomFiltersAboveTheBottomLevel(options)
	setBlockSizeOnEveryLevel(options)
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

func setBlockSizeOnEveryLevel(options *pebble.Options) {
	for level := range options.Levels {
		options.Levels[level].BlockSize = blockBytes
	}
}

func eventListenerFor(stalls WriteStallObserver) *pebble.EventListener {
	if stalls == nil {
		stalls = silentWriteStallObserver{}
	}

	return &pebble.EventListener{
		BackgroundError: reportBackgroundFailure,
		WriteStallBegin: func(info pebble.WriteStallBeginInfo) {
			reportWriteStallBegan(stalls, writeStallCauseOf(info.Reason))
		},
		WriteStallEnd: func() { reportWriteStallEnded(stalls) },
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

func (e *Engine) ProvisionMemberSetsBucket(_ vault.Name) error {
	return nil
}

func (e *Engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := e.commitInWriteSlot(ctx, fn); err != nil {
		return err
	}

	return e.syncCommits()
}

func (e *Engine) commitInWriteSlot(ctx context.Context, fn func(vault.EngineTxn) error) error {
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

	e.holdCommits()
	defer e.resumeCommits()

	if key, conflicted := e.history.keyWrittenAfter(begunAfter, write.footprint); conflicted {
		reportWriteConflicted(ctx, e.conflicts, key)

		return true, release(write.changes)
	}
	if err != nil {
		return false, errors.Join(err, release(write.changes))
	}

	return false, e.commit(write)
}

type stagedWrite struct {
	changes     *pebble.Batch
	writtenKeys sortedKeys
	footprint   *readFootprint
}

func (e *Engine) stage(fn func(vault.EngineTxn) error) (stagedWrite, error) {
	write := stagedWrite{changes: e.db.NewIndexedBatch(), footprint: newReadFootprint()}
	heldValues := heldValues{}
	amountChanges := newAmountChanges(write.changes, heldValues)
	memberSetChanges := newMemberSetChanges(write.changes, heldValues)
	if err := fn(&pebbleTxn{
		reader:           write.changes,
		pointReaders:     lookupPointReader{reader: write.changes},
		changes:          write.changes,
		amountChanges:    amountChanges,
		memberSetChanges: memberSetChanges,
		footprint:        write.footprint,
		heldValues:       heldValues,
		scans:            e.scans,
	}); err != nil {
		return write, err
	}
	if err := errors.Join(amountChanges.stageAll(), memberSetChanges.stageAll()); err != nil {
		return write, err
	}
	writtenKeys, err := writtenKeysOf(write.changes)
	if err != nil {
		return write, err
	}
	write.writtenKeys = writtenKeys

	return write, nil
}

func (e *Engine) holdCommits() { e.commitTurn <- struct{}{} }

func (e *Engine) resumeCommits() { <-e.commitTurn }

func (e *Engine) commit(write stagedWrite) error {
	return errors.Join(e.commitRecorded(write), release(write.changes))
}

func (e *Engine) commitRecorded(write stagedWrite) error {
	if err := write.changes.Commit(pebble.NoSync); err != nil {
		return commitFailureOf(err)
	}
	e.history.transactionCommitted(write.writtenKeys)

	return nil
}

func (e *Engine) updateExclusively(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.holdCommits()
	defer e.resumeCommits()

	begunAfter := e.history.transactionBegan()
	defer e.history.transactionEnded(begunAfter)

	reportExclusiveWrite(ctx, e.conflicts)
	write, err := e.stage(fn)
	if err != nil {
		return errors.Join(err, release(write.changes))
	}

	return e.commit(write)
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

func (e *Engine) syncCommits() error {
	return commitFailureOf(e.db.LogData(nil, pebble.Sync))
}

func (e *Engine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	snapshot := e.db.NewSnapshot()
	pointReaders := &iteratorPointReaders{reader: snapshot}

	if err := fn(&pebbleTxn{
		reader:           snapshot,
		pointReaders:     pointReaders,
		amountChanges:    &amountChanges{},
		memberSetChanges: &memberSetChanges{},
		scans:            e.scans,
	}); err != nil {
		return errors.Join(
			fmt.Errorf("read storage: %w", err),
			pointReaders.release(),
			release(snapshot),
		)
	}

	return errors.Join(pointReaders.release(), release(snapshot))
}

func (e *Engine) Close() error {
	e.surveyor.stop()
	if err := e.db.Close(); err != nil {
		return fmt.Errorf("close storage: %w", err)
	}

	return nil
}

func (e *Engine) QuotaBytes() int64 {
	return e.quotaBytes
}

func (e *Engine) Condition() EngineCondition {
	return engineConditionOf(e.db.Metrics(), e.limits, e.scans)
}

func (e *Engine) UsedBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	recordsDiskBytes, err := e.recordsDiskBytes()
	if err != nil {
		return 0, err
	}
	regionsDiskBytes, err := e.diskBytesWithin(amountsRegion, memberSetsRegion)
	if err != nil {
		return 0, err
	}

	return recordsDiskBytes + regionsDiskBytes, nil
}

func (e *Engine) recordsDiskBytes() (int64, error) {
	tallies, err := bucketTalliesOf(e.db)
	if err != nil {
		return 0, err
	}

	var diskBytes int64
	for bucket, tally := range tallies {
		diskBytes += e.surveyor.diskBytesOf(compressionratios.BucketTally{
			Prefix:    recordsRegionOf(bucket).prefix,
			Records:   tally.records,
			HeldBytes: tally.heldBytes,
		})
	}

	return diskBytes, nil
}

func (e *Engine) diskBytesWithin(regions ...keyspaceRegion) (int64, error) {
	var diskBytes int64
	for _, region := range regions {
		estimatedBytes, err := e.db.EstimateDiskUsage(region.boundsFor(vault.EveryKey()))
		if err != nil {
			return 0, fmt.Errorf("estimate disk bytes within region: %w", err)
		}
		diskBytes += signed(estimatedBytes)
	}

	return diskBytes, nil
}
