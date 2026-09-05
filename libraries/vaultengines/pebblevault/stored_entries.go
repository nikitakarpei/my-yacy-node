package pebblevault

import (
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type storedEntries struct {
	region keyspaceRegion
	reader pebble.Reader
}

func (e storedEntries) valueAt(key []byte) ([]byte, error) {
	value, valueHandle, err := e.reader.Get(e.region.absoluteKeyFrom(key))
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	held := make([]byte, len(value))
	copy(held, value)

	return held, release(valueHandle)
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

	return errors.Join(visitFound(found, e.region, fn), release(found))
}

func visitFound(
	found *pebble.Iterator,
	region keyspaceRegion,
	fn func(key, value []byte) (bool, error),
) error {
	for found.First(); found.Valid(); found.Next() {
		keep, err := fn(region.relativeKeyFrom(found.Key()), found.Value())
		if err != nil {
			return err
		}
		if !keep {
			return nil
		}
	}

	return nil
}
