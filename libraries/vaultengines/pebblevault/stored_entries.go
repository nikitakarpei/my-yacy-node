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
	footprint *readFootprint
}

func (e storedEntries) valueAt(key []byte) ([]byte, error) {
	absoluteKey := e.region.absoluteKeyFrom(key)
	e.footprint.addKey(absoluteKey)

	value, valueHandle, err := e.reader.Get(absoluteKey)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	heldValue := make([]byte, len(value))
	copy(heldValue, value)

	return heldValue, release(valueHandle)
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
