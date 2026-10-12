package pebblevault

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type storedEntries struct {
	region    keyspaceRegion
	reader    pebble.Reader
	changes   *pebble.Batch
	footprint *readFootprint
}

func (e storedEntries) valueAt(key []byte) ([]byte, error) {
	return e.valueAtAbsoluteKey(e.region.absoluteKeyFrom(key))
}

func (e storedEntries) valueAtAbsoluteKey(absoluteKey []byte) ([]byte, error) {
	if value, known := e.footprint.valueKnownAt(absoluteKey); known {
		return value, nil
	}

	value, err := e.storedValueAt(absoluteKey)
	if err != nil {
		return nil, err
	}
	e.footprint.addKeyHolding(absoluteKey, value)

	return value, nil
}

func (e storedEntries) storedValueAt(absoluteKey []byte) ([]byte, error) {
	value, valueHandle, err := e.reader.Get(absoluteKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	copiedValue := make([]byte, len(value))
	copy(copiedValue, value)

	return copiedValue, release(valueHandle)
}

func (e storedEntries) store(key, value []byte) (replacedValue []byte, err error) {
	absoluteKey := e.region.absoluteKeyFrom(key)
	replacedValue, err = e.valueAtAbsoluteKey(absoluteKey)
	if err != nil {
		return nil, err
	}
	if err := e.changes.Set(absoluteKey, value, pebble.NoSync); err != nil {
		return nil, fmt.Errorf("store: %w", err)
	}
	e.footprint.addKeyHolding(absoluteKey, value)

	return replacedValue, nil
}

func (e storedEntries) remove(key []byte) (removedValue []byte, err error) {
	absoluteKey := e.region.absoluteKeyFrom(key)
	removedValue, err = e.valueAtAbsoluteKey(absoluteKey)
	if err != nil || removedValue == nil {
		return nil, err
	}
	if err := e.changes.Delete(absoluteKey, pebble.NoSync); err != nil {
		return nil, fmt.Errorf("delete: %w", err)
	}
	e.footprint.addKeyHolding(absoluteKey, nil)

	return removedValue, nil
}

func (e storedEntries) merge(key, operand []byte) error {
	absoluteKey := e.region.absoluteKeyFrom(key)
	if err := e.changes.Merge(absoluteKey, operand, pebble.NoSync); err != nil {
		return fmt.Errorf("store amount: %w", err)
	}
	e.footprint.forgetValueAt(absoluteKey)

	return nil
}

func (e storedEntries) absoluteKeyFrom(key []byte) []byte {
	return e.region.absoluteKeyFrom(key)
}

func (e storedEntries) visit(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	firstIncluded, firstExcluded := e.region.boundsFor(keys)

	found, err := e.reader.NewIter(&pebble.IterOptions{
		LowerBound: firstIncluded,
		UpperBound: firstExcluded,
	})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	stoppedAt, err := visitFound(found, e.region, fn)
	e.footprint.addRange(firstIncluded, scanEndOf(stoppedAt, firstExcluded))

	return errors.Join(err, release(found))
}

func visitFound(
	found *pebble.Iterator,
	region keyspaceRegion,
	fn func(key, value []byte) (bool, error),
) (stoppedAt []byte, err error) {
	for found.First(); found.Valid(); found.Next() {
		keep, err := fn(region.relativeKeyFrom(found.Key()), found.Value())
		if err != nil || !keep {
			return found.Key(), err
		}
	}

	return nil, nil
}

func scanEndOf(stoppedAt, firstExcluded []byte) []byte {
	if stoppedAt == nil {
		return firstExcluded
	}

	return firstKeyAfter(stoppedAt)
}

func firstKeyAfter(key []byte) []byte {
	return append(bytes.Clone(key), 0)
}
