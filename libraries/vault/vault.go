// Package vault lends callers typed, transactional views over named buckets.
// Records buckets back a Collection, a Set, a ValueIndex or a ChangeIndex;
// amounts buckets hold the sum of changes raised and lowered blind, read as never below
// zero, or a CountIndex; member sets buckets back a MemberSetIndex over a MemberEncoding.
// Expiring records buckets back an ExpiringCollection: a record reads as absent
// once its lifetime after its last write ends, and vault removes it with its index
// entries later. Each transaction run reads the vault clock once, as its time.
// Only vault adds to an index, in step with the collection it is attached to. An Engine
// stores the buckets, keeps each write whole, may rerun its closure, and counts
// bytes to a quota, and never removes an expiring record unasked. Transactions are
// serializable: each reads one committed state, scans included, and every outcome is one
// that some serial order of the transactions gives; an engine runs a write again where
// that order would otherwise break. Blind changes of one amount never make each other
// run again. A write that keeps running again is at last run alone, so every write ends
// within a bounded number of runs.
package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Name string

var (
	errVaultClosed  = errors.New("vault closed")
	errEngineAbsent = errors.New("vault needs a storage engine")
	errClockAbsent  = errors.New("vault needs a clock")
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Vault struct {
	engine             Engine
	observer           TransactionObserver
	clock              Clock
	removals           expiryRemovals
	mu                 sync.Mutex
	registered         map[Name]bucketKind
	projectionVersions func() (projectionVersions, error)
}

func New(engine Engine, observer TransactionObserver, clock Clock) (*Vault, error) {
	if engine == nil {
		return nil, errEngineAbsent
	}
	if clock == nil {
		return nil, errClockAbsent
	}
	if observer == nil {
		observer = silentObserver{}
	}
	storage := &Vault{
		engine:     engine,
		observer:   observer,
		clock:      clock,
		registered: map[Name]bucketKind{},
	}
	storage.projectionVersions = sync.OnceValues(storage.registerProjectionVersions)

	return storage, nil
}

func (v *Vault) Close() error {
	if v == nil || v.engine == nil {
		return nil
	}

	v.removals.stop()
	err := v.engine.Close()
	v.engine = nil
	if err != nil {
		return fmt.Errorf("close storage: %w", err)
	}

	return nil
}

func (v *Vault) QuotaBytes() int64 {
	if v == nil || v.engine == nil {
		return 0
	}

	return v.engine.QuotaBytes()
}

func (v *Vault) UsedBytes(ctx context.Context) (int64, error) {
	if v == nil || v.engine == nil {
		return 0, errVaultClosed
	}

	used, err := v.engine.UsedBytes(ctx)
	if err != nil {
		return 0, fmt.Errorf("measure used bytes: %w", err)
	}

	return used, nil
}

func (v *Vault) AtCapacity(ctx context.Context) (bool, error) {
	if v == nil || v.engine == nil {
		return false, errVaultClosed
	}
	if v.engine.QuotaBytes() <= 0 {
		return false, nil
	}

	used, err := v.UsedBytes(ctx)
	if err != nil {
		return false, err
	}

	return used >= v.engine.QuotaBytes(), nil
}
