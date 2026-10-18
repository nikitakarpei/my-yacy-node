package vault

import (
	"context"
	"slices"
)

type PairKeyIndex[K any] interface {
	projectionBucket
	keyLayout() KeyLayout[K]
	secondPartOf(key K) string
	storedKeysWithSecondPartOf(tx *Txn, key K) (storedKeys[K], error)
	keyAdded(tx *Txn, key K) error
	keyRemoved(tx *Txn, key K) error
}

func (v *Vault) RegisterIndexedCollection[K, V any](
	ctx context.Context,
	bucket Name,
	index PairKeyIndex[K],
	values ValueCodec[V],
	projections Projections[K, V],
) (*Collection[K, V], error) {
	if projections == nil {
		projections = ProjectedAs[K, V](func(key K, _ V) K { return key })
	}

	return v.registerProjected(ctx, &Collection[K, V]{
		name:        bucket,
		keys:        index.keyLayout(),
		values:      values,
		projections: indexedByPairKey[K, V]{index: index, projections: projections},
		index:       index,
	})
}

type indexedByPairKey[K, V any] struct {
	index       PairKeyIndex[K]
	projections Projections[K, V]
}

func (p indexedByPairKey[K, V]) recordAdded(tx *Txn, key K, value storedForm[V]) error {
	if p.index != nil {
		if err := p.index.keyAdded(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordAdded(tx, key, value)
}

func (p indexedByPairKey[K, V]) recordRemoved(tx *Txn, key K, value storedForm[V]) error {
	if p.index != nil {
		if err := p.index.keyRemoved(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordRemoved(tx, key, value)
}

func (p indexedByPairKey[K, V]) recordReplaced(
	tx *Txn,
	key K,
	previous, current storedForm[V],
) error {
	return p.projections.recordReplaced(tx, key, previous, current)
}

func (p indexedByPairKey[K, V]) buckets() []projectionBucket {
	return append([]projectionBucket{p.index}, p.projections.buckets()...)
}

func (p indexedByPairKey[K, V]) narrowedTo(buckets []projectionBucket) Projections[K, V] {
	narrowed := indexedByPairKey[K, V]{projections: p.projections.narrowedTo(buckets)}
	if slices.Contains(buckets, projectionBucket(p.index)) {
		narrowed.index = p.index
	}

	return narrowed
}
