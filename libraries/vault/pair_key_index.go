package vault

import "slices"

type PairKeyIndex[K any] interface {
	projectionBucket
	keyLayout() KeyLayout[K]
	secondPartOf(key K) string
	storedKeysWithSecondPartOf(tx *Txn, key K) (storedKeys[K], error)
	keyAdded(tx *Txn, key K) error
	keyRemoved(tx *Txn, key K) error
}

type indexedByPairKey[K, V any] struct {
	index       PairKeyIndex[K]
	projections Projections[K, V]
}

func (p indexedByPairKey[K, V]) recordAdded(tx *Txn, key K, added storedValue[V]) error {
	if p.index != nil {
		if err := p.index.keyAdded(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordAdded(tx, key, added)
}

func (p indexedByPairKey[K, V]) recordRemoved(tx *Txn, key K, removed storedValue[V]) error {
	if p.index != nil {
		if err := p.index.keyRemoved(tx, key); err != nil {
			return err
		}
	}

	return p.projections.recordRemoved(tx, key, removed)
}

func (p indexedByPairKey[K, V]) recordReplaced(
	tx *Txn,
	key K,
	previous, current storedValue[V],
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
