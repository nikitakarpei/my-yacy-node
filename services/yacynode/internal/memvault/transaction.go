package memvault

import (
	"sort"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
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

func (b memBucket) Get(key []byte) []byte {
	value, ok := b.entries[string(key)]
	if !ok {
		return nil
	}

	return value
}

func (b memBucket) Put(key []byte, value []byte) error {
	b.entries[string(key)] = copyValue(value)

	return nil
}

func (b memBucket) Delete(key []byte) error {
	delete(b.entries, string(key))

	return nil
}

func (b memBucket) Scan(keys vaultkey.KeyRange, fn func(key, value []byte) (bool, error)) error {
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

func orderedKeysOf(entries map[string][]byte, keys vaultkey.KeyRange) []string {
	ordered := make([]string, 0, len(entries))
	for key := range entries {
		if keys.Contains([]byte(key)) {
			ordered = append(ordered, key)
		}
	}
	sort.Strings(ordered)

	return ordered
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
