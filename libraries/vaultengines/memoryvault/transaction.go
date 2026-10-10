package memoryvault

import (
	"maps"
	"sort"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memTxn struct {
	records  map[vault.Name]map[string][]byte
	amounts  map[vault.Name]map[string]int
	writable bool
}

func (t memTxn) Writable() bool { return t.writable }

func (t memTxn) Records(name vault.Name) vault.EngineRecords {
	return memRecords{entries: t.records[name]}
}

func (t memTxn) Amounts(name vault.Name) vault.EngineAmounts {
	return memAmounts{entries: t.amounts[name]}
}

type memRecords struct {
	entries map[string][]byte
}

func (b memRecords) Get(key []byte) ([]byte, error) {
	value, ok := b.entries[string(key)]
	if !ok {
		return nil, nil
	}

	return value, nil
}

func (b memRecords) Put(key []byte, record []byte) ([]byte, error) {
	replacedRecord := b.entries[string(key)]
	b.entries[string(key)] = copyValue(record)

	return replacedRecord, nil
}

func (b memRecords) Delete(key []byte) ([]byte, error) {
	deletedRecord, found := b.entries[string(key)]
	if !found {
		return nil, nil
	}
	delete(b.entries, string(key))

	return deletedRecord, nil
}

func (b memRecords) Len() (int, error) {
	return len(b.entries), nil
}

func (b memRecords) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	for _, key := range orderedKeysOf(b.entries, keys) {
		keep, err := fn([]byte(key), b.entries[key])
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

func snapshotOfRecords(source map[vault.Name]map[string][]byte) map[vault.Name]map[string][]byte {
	copied := make(map[vault.Name]map[string][]byte, len(source))
	for name, bucket := range source {
		entries := make(map[string][]byte, len(bucket))
		for key, value := range bucket {
			entries[key] = copyValue(value)
		}
		copied[name] = entries
	}

	return copied
}

func snapshotOfAmounts(source map[vault.Name]map[string]int) map[vault.Name]map[string]int {
	copied := make(map[vault.Name]map[string]int, len(source))
	for name, amounts := range source {
		copied[name] = maps.Clone(amounts)
	}

	return copied
}

func copyValue(value []byte) []byte {
	copied := make([]byte, len(value))
	copy(copied, value)

	return copied
}
