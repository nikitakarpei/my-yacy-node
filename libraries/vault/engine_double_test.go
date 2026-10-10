package vault_test

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type doubleEngine struct {
	records    map[vault.Name]map[string][]byte
	amounts    map[vault.Name]map[string]int
	quotaBytes int64
}

func newDoubleEngine() *doubleEngine {
	return &doubleEngine{
		records: map[vault.Name]map[string][]byte{},
		amounts: map[vault.Name]map[string]int{},
	}
}

func openDouble() (*vault.Vault, error) {
	v, err := vault.New(newDoubleEngine(), nil)
	if err != nil {
		return nil, fmt.Errorf("new vault: %w", err)
	}

	return v, nil
}

func (e *doubleEngine) ProvisionRecordsBucket(name vault.Name) error {
	if _, ok := e.records[name]; !ok {
		e.records[name] = map[string][]byte{}
	}

	return nil
}

func (e *doubleEngine) ProvisionAmountsBucket(name vault.Name) error {
	if _, ok := e.amounts[name]; !ok {
		e.amounts[name] = map[string]int{}
	}

	return nil
}

func (e *doubleEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	staged := doubleTxn{
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

func (e *doubleEngine) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	return fn(doubleTxn{records: e.records, amounts: e.amounts, writable: false})
}

func (e *doubleEngine) Close() error {
	e.records = nil

	return nil
}

func (e *doubleEngine) QuotaBytes() int64 { return e.quotaBytes }

func (e *doubleEngine) UsedBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("context: %w", err)
	}

	var used int64
	for _, bucket := range e.records {
		for key, value := range bucket {
			used += int64(len(key) + len(value))
		}
	}

	return used, nil
}

type doubleTxn struct {
	records  map[vault.Name]map[string][]byte
	amounts  map[vault.Name]map[string]int
	writable bool
}

func (t doubleTxn) Writable() bool { return t.writable }

func (t doubleTxn) Records(name vault.Name) vault.EngineRecords {
	return doubleBucket{entries: t.records[name]}
}

func (t doubleTxn) Amounts(name vault.Name) vault.EngineAmounts {
	return doubleAmounts{entries: t.amounts[name]}
}

type doubleAmounts struct {
	entries map[string]int
}

func (a doubleAmounts) Get(key []byte) (int, error) {
	return a.entries[string(key)], nil
}

func (a doubleAmounts) Raise(key []byte, by uint) error {
	a.entries[string(key)] += int(by)

	return nil
}

func (a doubleAmounts) Lower(key []byte, by uint) error {
	a.entries[string(key)] -= int(by)
	if a.entries[string(key)] <= 0 {
		delete(a.entries, string(key))
	}

	return nil
}

func (a doubleAmounts) Scan(
	keys vault.KeyRange,
	fn func(key []byte, amount int) (bool, error),
) error {
	ordered := make([]string, 0, len(a.entries))
	firstIncluded, firstExcluded := keys.Bounds()
	for key := range a.entries {
		if isWithinBounds(key, firstIncluded, firstExcluded) {
			ordered = append(ordered, key)
		}
	}
	sort.Strings(ordered)

	for _, key := range ordered {
		keep, err := fn([]byte(key), a.entries[key])
		if err != nil || !keep {
			return err
		}
	}

	return nil
}

type doubleBucket struct {
	entries map[string][]byte
}

func (b doubleBucket) Get(key []byte) ([]byte, error) {
	value, ok := b.entries[string(key)]
	if !ok {
		return nil, nil
	}

	return copyBytes(value), nil
}

func (b doubleBucket) Put(key []byte, record []byte) ([]byte, error) {
	replacedRecord := b.entries[string(key)]
	b.entries[string(key)] = copyBytes(record)

	return replacedRecord, nil
}

func (b doubleBucket) Delete(key []byte) ([]byte, error) {
	deletedRecord, found := b.entries[string(key)]
	if !found {
		return nil, nil
	}
	delete(b.entries, string(key))

	return deletedRecord, nil
}

func (b doubleBucket) Len() (int, error) {
	return len(b.entries), nil
}

func (b doubleBucket) Scan(
	keys vault.KeyRange,
	fn func(key, value []byte) (bool, error),
) error {
	for _, key := range orderedKeysOf(b.entries, keys) {
		keep, err := fn([]byte(key), copyBytes(b.entries[key]))
		if err != nil {
			return err
		}
		if !keep {
			return nil
		}
	}

	return nil
}

func orderedKeysOf(entries map[string][]byte, keys vault.KeyRange) []string {
	firstIncluded, firstExcluded := keys.Bounds()

	ordered := make([]string, 0, len(entries))
	for key := range entries {
		if isWithinBounds(key, firstIncluded, firstExcluded) {
			ordered = append(ordered, key)
		}
	}
	sort.Strings(ordered)

	return ordered
}

func isWithinBounds(key string, firstIncluded, firstExcluded []byte) bool {
	if key < string(firstIncluded) {
		return false
	}

	return firstExcluded == nil || key < string(firstExcluded)
}

func snapshotOfRecords(
	source map[vault.Name]map[string][]byte,
) map[vault.Name]map[string][]byte {
	staged := make(map[vault.Name]map[string][]byte, len(source))
	for name, bucket := range source {
		entries := make(map[string][]byte, len(bucket))
		for key, value := range bucket {
			entries[key] = copyBytes(value)
		}
		staged[name] = entries
	}

	return staged
}

func snapshotOfAmounts(source map[vault.Name]map[string]int) map[vault.Name]map[string]int {
	staged := make(map[vault.Name]map[string]int, len(source))
	for name, amounts := range source {
		staged[name] = maps.Clone(amounts)
	}

	return staged
}

func copyBytes(value []byte) []byte {
	out := make([]byte, len(value))
	copy(out, value)

	return out
}

func (e *doubleEngine) plant(bucket vault.Name, key, record []byte) {
	if _, ok := e.records[bucket]; !ok {
		e.records[bucket] = map[string][]byte{}
	}
	e.records[bucket][string(key)] = record
}
