// Package vault lends callers typed, transactional views over named buckets.
// Records buckets back a Collection, a Set or a ValueIndex; amounts buckets
// hold numbers raised and lowered blind, or a CountIndex. Only vault writes an
// index, in step with the collection it follows. An Engine stores the buckets,
// keeps each write whole, may rerun its closure, and counts bytes to a quota.
package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Name string

var (
	errVaultClosed  = errors.New("vault closed")
	errEngineAbsent = errors.New("vault needs a storage engine")
)

type Vault struct {
	engine             Engine
	observer           TransactionObserver
	mu                 sync.Mutex
	registered         map[Name]bucketKind
	projectionVersions func() (*Collection[Name, string], error)
}

func New(engine Engine, observer TransactionObserver) (*Vault, error) {
	if engine == nil {
		return nil, errEngineAbsent
	}
	if observer == nil {
		observer = silentObserver{}
	}
	storage := &Vault{
		engine:     engine,
		observer:   observer,
		registered: map[Name]bucketKind{},
	}
	storage.projectionVersions = sync.OnceValues(storage.registerProjectionVersions)

	return storage, nil
}

func (v *Vault) Close() error {
	if v == nil || v.engine == nil {
		return nil
	}

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
