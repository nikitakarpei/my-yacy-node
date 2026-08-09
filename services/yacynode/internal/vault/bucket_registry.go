package vault

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
)

func Register[K, V any](
	v *Vault,
	bucket Name,
	keys KeyCodec[K],
	values ValueCodec[V],
) (*Collection[K, V], error) {
	if v == nil || v.engine == nil {
		return nil, errVaultClosed
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if _, dup := v.registered[bucket]; dup {
		return nil, fmt.Errorf("%w: %s", errDuplicateBucket, bucket)
	}

	if err := v.engine.Provision(bucket); err != nil {
		return nil, fmt.Errorf("register bucket %s: %w", bucket, err)
	}

	v.registered[bucket] = struct{}{}

	return &Collection[K, V]{name: bucket, keys: keys, values: values}, nil
}

func (v *Vault) EntriesByCollection(ctx context.Context) (map[Name]int, error) {
	if v == nil || v.engine == nil {
		return nil, errVaultClosed
	}

	collections := v.registeredCollections()

	var entries map[Name]int

	if err := v.view(ctx, func(tx *Txn) error {
		lengths, err := lengthsOf(tx, collections)
		entries = lengths

		return err
	}); err != nil {
		return nil, fmt.Errorf("read collection entries: %w", err)
	}

	return entries, nil
}

func (v *Vault) registeredCollections() []Name {
	v.mu.Lock()
	defer v.mu.Unlock()

	collections := make([]Name, 0, len(v.registered))
	for bucket := range v.registered {
		collections = append(collections, bucket)
	}

	return collections
}

func lengthsOf(tx *Txn, collections []Name) (map[Name]int, error) {
	lengths := make(map[Name]int, len(collections))
	for _, bucket := range collections {
		length, err := readLength(tx, bucket)
		if err != nil {
			return nil, fmt.Errorf("length of %s: %w", bucket, err)
		}
		lengths[bucket] = length
	}

	return lengths, nil
}

func readLength(tx *Txn, bucket Name) (int, error) {
	return decodeLength(tx.etx.Bucket(lengthBucket).Get([]byte(bucket)))
}

func adjustLength(tx *Txn, bucket Name, delta int) error {
	lengths := tx.etx.Bucket(lengthBucket)
	current, err := decodeLength(lengths.Get([]byte(bucket)))
	if err != nil {
		return err
	}

	return putLength(lengths, []byte(bucket), max(current+delta, 0))
}

func decodeLength(raw []byte) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if len(raw) != 8 {
		return 0, fmt.Errorf("bad length counter: %d bytes", len(raw))
	}

	n := binary.BigEndian.Uint64(raw)
	if n > uint64(int(^uint(0)>>1)) {
		return 0, errors.New("length counter overflow")
	}

	return int(n), nil
}

func putLength(lengths EngineBucket, key []byte, n int) error {
	if n < 0 {
		return errors.New("negative length counter")
	}

	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], uint64(n))
	if err := lengths.Put(key, raw[:]); err != nil {
		return fmt.Errorf("store length counter: %w", err)
	}

	return nil
}
