package memoryvault

import (
	"sort"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memTxn struct {
	buckets  map[vault.Name]map[string][]byte
	writable bool
}

func (t memTxn) Writable() bool { return t.writable }

func (t memTxn) Bucket(name vault.Name) vault.EngineBucket {
	return memBucket{entries: t.buckets[name]}
}

type memBucket struct {
	entries map[string][]byte
}

func (b memBucket) Get(key []byte) ([]byte, error) {
	value, ok := b.entries[string(key)]
	if !ok {
		return nil, nil
	}

	return value, nil
}

func (b memBucket) Put(key []byte, value []byte) error {
	b.entries[string(key)] = copyValue(value)

	return nil
}

func (b memBucket) Delete(key []byte) (bool, error) {
	if _, present := b.entries[string(key)]; !present {
		return false, nil
	}
	delete(b.entries, string(key))

	return true, nil
}

func (b memBucket) Len() (int, error) {
	return len(b.entries), nil
}

func (b memBucket) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
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

func snapshot(source map[vault.Name]map[string][]byte) map[vault.Name]map[string][]byte {
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

func copyValue(value []byte) []byte {
	copied := make([]byte, len(value))
	copy(copied, value)

	return copied
}
