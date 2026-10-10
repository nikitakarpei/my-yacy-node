// Package vault lends callers typed, transactional views over named buckets.
// A records bucket holds one record per key and backs a Collection or a Set.
// An amounts bucket holds one number per key that writers raise and lower
// without reading it. An Engine stores the buckets, keeps each write whole and
// can run its closure again, and counts the bytes used against its quota.
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
	engine     Engine
	observer   TransactionObserver
	mu         sync.Mutex
	registered map[Name]bucketKind
}

func New(engine Engine, observer TransactionObserver) (*Vault, error) {
	if engine == nil {
		return nil, errEngineAbsent
	}
	if observer == nil {
		observer = silentObserver{}
	}
	return &Vault{
		engine:     engine,
		observer:   observer,
		registered: map[Name]bucketKind{},
	}, nil
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
