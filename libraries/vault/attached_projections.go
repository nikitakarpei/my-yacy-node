package vault

import (
	"bytes"
	"fmt"
)

type attachedProjections[K, V any] struct {
	projections Projections[K, V]
	codec       recordCodec[K, V]
}

func (p attachedProjections[K, V]) recordStored(
	tx *Txn,
	key K,
	stored storedValue[V],
	replacedRecord []byte,
) error {
	if p.projections == nil {
		return nil
	}
	if replacedRecord == nil {
		return p.projections.recordAdded(tx, key, stored)
	}
	replaced, err := p.codec.storedValueFrom(replacedRecord)
	if err != nil {
		return fmt.Errorf("replaced record: %w", err)
	}

	return p.projections.recordReplaced(tx, key, replaced, stored)
}

func (p attachedProjections[K, V]) recordRemoved(tx *Txn, key K, removedRecord []byte) error {
	if p.projections == nil || removedRecord == nil {
		return nil
	}
	removed, err := p.codec.storedValueFrom(removedRecord)
	if err != nil {
		return fmt.Errorf("removed record: %w", err)
	}

	return p.projections.recordRemoved(tx, key, removed)
}

func (p attachedProjections[K, V]) buckets() []projectionBucket {
	if p.projections == nil {
		return nil
	}

	return p.projections.buckets()
}

type storedRecords interface {
	Scan(keys KeyRange, fn func(key, record []byte) (bool, error)) error
	Len() (int, error)
}

func (p attachedProjections[K, V]) projectStoredRecordsFrom(
	tx *Txn,
	records storedRecords,
	firstKey []byte,
	limit int,
	stale []projectionBucket,
) (nextKey []byte, err error) {
	staleProjections := p.projections.narrowedTo(stale)
	projected := 0
	err = records.Scan(KeyRange{firstIncluded: firstKey}, func(key, record []byte) (bool, error) {
		if projected == limit {
			nextKey = bytes.Clone(key)

			return false, nil
		}
		projected++

		return true, p.projectStoredRecord(tx, key, record, staleProjections)
	})

	return nextKey, err
}

func (p attachedProjections[K, V]) projectStoredRecord(
	tx *Txn,
	key, record []byte,
	staleProjections Projections[K, V],
) error {
	decodedKey, err := p.codec.keyFrom(key)
	if err != nil {
		return err
	}
	stored, err := p.codec.storedValueFrom(record)
	if err != nil {
		return err
	}

	return staleProjections.recordAdded(tx, decodedKey, stored)
}
