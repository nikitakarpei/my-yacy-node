package vault

import (
	"context"
	"maps"
)

const fewestRecordsPerSecondPartRead = 8

type IndexedCollection[K, V any] struct {
	*Collection[K, V]
	index PairKeyIndex[K]
}

type KeyedValue[K, V any] struct {
	Key   K
	Value V
}

func (v *Vault) RegisterIndexedCollection[K, V any](
	ctx context.Context,
	bucket Name,
	index PairKeyIndex[K],
	values ValueCodec[V],
	projections Projections[K, V],
) (*IndexedCollection[K, V], error) {
	if projections == nil {
		projections = ProjectedAs[K, V](func(key K, _ V) K { return key })
	}

	collection, err := v.registerProjected(ctx, &Collection[K, V]{
		name:        bucket,
		keys:        index.keyLayout(),
		values:      values,
		projections: indexedByPairKey[K, V]{index: index, projections: projections},
	})
	if err != nil {
		return nil, err
	}

	return &IndexedCollection[K, V]{Collection: collection, index: index}, nil
}

func (c *IndexedCollection[K, V]) PutEach(tx *Txn, keyedValues []KeyedValue[K, V]) error {
	absentPositions, err := c.absentPositionsAmong(tx, keyedValues)
	if err != nil {
		return err
	}
	for position, keyedValue := range keyedValues {
		if _, isAbsent := absentPositions[position]; isAbsent {
			err = c.storeAbsentRecord(tx, keyedValue.Key, keyedValue.Value)
		} else {
			_, err = c.storeRecord(tx, keyedValue.Key, keyedValue.Value)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *IndexedCollection[K, V]) absentPositionsAmong(
	tx *Txn,
	keyedValues []KeyedValue[K, V],
) (map[int]struct{}, error) {
	storedKeysPerSecondPart, err := c.storedKeysWithin(
		tx,
		c.crowdedSecondPartsAmong(keyedValues),
	)
	if err != nil {
		return nil, err
	}
	absentPositions := make(map[int]struct{})
	for position, keyedValue := range keyedValues {
		keys, isRead := storedKeysPerSecondPart[c.index.secondPartOf(keyedValue.Key)]
		if isRead && !keys.holds(keyedValue.Key) {
			absentPositions[position] = struct{}{}
			keys.add(keyedValue.Key)
		}
	}

	return absentPositions, nil
}

func (c *IndexedCollection[K, V]) crowdedSecondPartsAmong(
	keyedValues []KeyedValue[K, V],
) map[string]K {
	amountPerSecondPart := make(map[string]int)
	keyPerSecondPart := make(map[string]K)
	for _, keyedValue := range keyedValues {
		secondPart := c.index.secondPartOf(keyedValue.Key)
		amountPerSecondPart[secondPart]++
		keyPerSecondPart[secondPart] = keyedValue.Key
	}
	maps.DeleteFunc(keyPerSecondPart, func(secondPart string, _ K) bool {
		return amountPerSecondPart[secondPart] < fewestRecordsPerSecondPartRead
	})

	return keyPerSecondPart
}

func (c *IndexedCollection[K, V]) storedKeysWithin(
	tx *Txn,
	keyPerSecondPart map[string]K,
) (map[string]storedKeys[K], error) {
	storedKeysPerSecondPart := make(map[string]storedKeys[K], len(keyPerSecondPart))
	for secondPart, key := range keyPerSecondPart {
		keys, err := c.index.storedKeysWithSecondPartOf(tx, key)
		if err != nil {
			return nil, err
		}
		storedKeysPerSecondPart[secondPart] = keys
	}

	return storedKeysPerSecondPart, nil
}
