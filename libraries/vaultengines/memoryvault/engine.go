// Package memoryvault is the in-memory implementation of the vault Engine. It keeps
// every bucket in process memory, owns no file, and survives only as long as the
// process. It backs tests and any deployment that cannot reach a filesystem.
package memoryvault

import (
	"context"
	"fmt"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const amountBytes = 8

type engine struct {
	mutex      sync.RWMutex
	records    map[vault.Name]map[string][]byte
	amounts    map[vault.Name]map[string]int
	quotaBytes int64
}

func Open(quotaBytes int64, observer vault.TransactionObserver) (*vault.Vault, error) {
	vaulted, err := vault.New(OpenEngine(quotaBytes), observer)
	if err != nil {
		return nil, fmt.Errorf("initialize storage: %w", err)
	}

	return vaulted, nil
}

func OpenEngine(quotaBytes int64) vault.Engine {
	return &engine{
		records:    map[vault.Name]map[string][]byte{},
		amounts:    map[vault.Name]map[string]int{},
		quotaBytes: quotaBytes,
	}
}

func (e *engine) ProvisionRecordsBucket(name vault.Name) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, ok := e.records[name]; !ok {
		e.records[name] = map[string][]byte{}
	}

	return nil
}

func (e *engine) ProvisionAmountsBucket(name vault.Name) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, ok := e.amounts[name]; !ok {
		e.amounts[name] = map[string]int{}
	}

	return nil
}

func (e *engine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	staged := memTxn{
		records:  snapshotOfRecords(e.records),
		amounts:  snapshotOfAmounts(e.amounts),
		writable: true,
	}
	if err := fn(staged); err != nil {
		return err
	}
	e.records = staged.records
	e.amounts = staged.amounts

	return nil
}

func (e *engine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	return fn(memTxn{records: e.records, amounts: e.amounts, writable: false})
}

func (e *engine) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.records = nil
	e.amounts = nil

	return nil
}

func (e *engine) QuotaBytes() int64 {
	return e.quotaBytes
}

func (e *engine) UsedBytes(ctx context.Context) (int64, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	var used int64
	for _, bucket := range e.records {
		for key, value := range bucket {
			used += int64(len(key) + len(value))
		}
	}
	for _, amounts := range e.amounts {
		for key := range amounts {
			used += int64(len(key) + amountBytes)
		}
	}

	return used, nil
}
