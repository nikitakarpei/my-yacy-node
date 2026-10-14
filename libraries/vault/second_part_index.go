package vault

import (
	"context"
	"fmt"
	"slices"
)

const secondPartIndexVersion = "1"

type SecondPartIndex[A, B, K any] struct {
	projectionRegistration
	parts   PairKeyParts[A, B]
	swapped PairKeyParts[B, A]
	partsOf func(K) (A, B)
	keyFrom func(A, B) K
}

func (v *Vault) RegisterSecondPartIndex[A, B, K any](
	bucket Name,
	parts PairKeyParts[A, B],
	partsOf func(K) (A, B),
	keyFrom func(A, B) K,
) (*SecondPartIndex[A, B, K], error) {
	if err := v.provisionRecordsBucket(bucket); err != nil {
		return nil, err
	}

	return &SecondPartIndex[A, B, K]{
		projectionRegistration: projectionRegistration{
			bucket:  bucket,
			version: secondPartIndexVersion,
		},
		parts:   parts,
		swapped: PairKey(parts.second, parts.first),
		partsOf: partsOf,
		keyFrom: keyFrom,
	}, nil
}

func (v *Vault) RegisterIndexedCollection[A, B, K, V any](
	ctx context.Context,
	bucket Name,
	index *SecondPartIndex[A, B, K],
	values ValueCodec[V],
	projections Projections[K, V],
) (*Collection[K, V], error) {
	if projections == nil {
		projections = ProjectedAs[K, V](func(key K, _ V) K { return key })
	}

	return v.registerProjected(ctx, &Collection[K, V]{
		name:        bucket,
		keys:        index.parts.KeyLayoutFor(index.partsOf, index.keyFrom),
		values:      values,
		projections: indexedBySecondPart[A, B, K, V]{index: index, projections: projections},
		secondParts: index,
	})
}

func (i *SecondPartIndex[A, B, K]) Scan(
	tx *Txn,
	second B,
	fn func(K) (bool, error),
) error {
	if err := i.refuseUnfollowed(); err != nil {
		return err
	}

	return i.visitKeysWith(tx, second, func(first A, second B) (bool, error) {
		return fn(i.keyFrom(first, second))
	})
}

func (i *SecondPartIndex[A, B, K]) visitKeysWith(
	tx *Txn,
	second B,
	visit func(first A, second B) (bool, error),
) error {
	if err := tx.etx.Records(i.bucket).Scan(
		i.swapped.KeysWithFirst(second),
		func(entryKey, _ []byte) (bool, error) {
			second, first, err := i.swapped.PartsOf(entryKey)
			if err != nil {
				return false, fmt.Errorf("decode %s entry: %w", i.bucket, err)
			}

			return visit(first, second)
		},
	); err != nil {
		return fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return nil
}

func (i *SecondPartIndex[A, B, K]) secondPartOf(key K) string {
	_, second := i.partsOf(key)

	return string(i.parts.second.encodingOf(second))
}

func (i *SecondPartIndex[A, B, K]) storedNeighboursOf(
	tx *Txn,
	key K,
) (map[string]struct{}, error) {
	_, second := i.partsOf(key)
	storedKeys := make(map[string]struct{})
	err := i.visitKeysWith(tx, second, func(first A, second B) (bool, error) {
		storedKeys[string(i.parts.Key(first, second).encoded)] = struct{}{}

		return true, nil
	})

	return storedKeys, err
}

func (i *SecondPartIndex[A, B, K]) keyAdded(tx *Txn, key K) error {
	if err := tx.etx.Records(i.bucket).PutAbsent(i.entryKeyOf(key), recordFrom(nil)); err != nil {
		return fmt.Errorf("add %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *SecondPartIndex[A, B, K]) entryKeyOf(key K) []byte {
	first, second := i.partsOf(key)

	return i.swapped.Key(second, first).encoded
}

func (i *SecondPartIndex[A, B, K]) keyRemoved(tx *Txn, key K) error {
	return i.removeEntry(tx, i.entryKeyOf(key))
}

func (i *SecondPartIndex[A, B, K]) removeEntry(tx *Txn, entryKey []byte) error {
	if err := tx.etx.Records(i.bucket).DeleteHeld(entryKey, recordFrom(nil)); err != nil {
		return fmt.Errorf("remove %s entry: %w", i.bucket, err)
	}

	return nil
}

func (i *SecondPartIndex[A, B, K]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	entryKeys, nextKey, err := recordKeysFrom(tx, i.bucket, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for _, entryKey := range entryKeys {
		if err := i.removeEntry(tx, entryKey); err != nil {
			return nil, err
		}
	}

	return nextKey, nil
}

type indexedBySecondPart[A, B, K, V any] struct {
	index       *SecondPartIndex[A, B, K]
	projections Projections[K, V]
}

func (p indexedBySecondPart[A, B, K, V]) recordAdded(tx *Txn, key K, value storedForm[V]) error {
	if p.index != nil {
		if err := p.index.keyAdded(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordAdded(tx, key, value)
}

func (p indexedBySecondPart[A, B, K, V]) recordRemoved(
	tx *Txn,
	key K,
	value storedForm[V],
) error {
	if p.index != nil {
		if err := p.index.keyRemoved(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordRemoved(tx, key, value)
}

func (p indexedBySecondPart[A, B, K, V]) recordReplaced(
	tx *Txn,
	key K,
	previous, current storedForm[V],
) error {
	return p.projections.recordReplaced(tx, key, previous, current)
}

func (p indexedBySecondPart[A, B, K, V]) buckets() []projectionBucket {
	return append([]projectionBucket{p.index}, p.projections.buckets()...)
}

func (p indexedBySecondPart[A, B, K, V]) narrowedTo(
	buckets []projectionBucket,
) Projections[K, V] {
	narrowed := indexedBySecondPart[A, B, K, V]{projections: p.projections.narrowedTo(buckets)}
	if slices.Contains(buckets, projectionBucket(p.index)) {
		narrowed.index = p.index
	}

	return narrowed
}
