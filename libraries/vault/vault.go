// Package vault lends each caller a transactionally-scoped, codec-typed
// collection over its own ordered byte buckets. It defines the storage seam: an
// Engine driver contract that a storage medium implements, and the
// Collection/Txn surface a caller speaks. It encodes each key so that the byte
// order of the key equals the domain order of its parts. No storage-medium type
// appears on its exported surface.
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
	registered map[Name]struct{}
}

func New(engine Engine, observer TransactionObserver) (*Vault, error) {
	if engine == nil {
		return nil, errEngineAbsent
	}
	if observer == nil {
		observer = silentObserver{}
	}
	if err := engine.Provision(lengthBucket); err != nil {
		return nil, fmt.Errorf("provision length bucket: %w", err)
	}

	return &Vault{
		engine:     engine,
		observer:   observer,
		registered: map[Name]struct{}{},
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
