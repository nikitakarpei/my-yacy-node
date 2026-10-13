package vault

import (
	"bytes"
	"fmt"
)

type CountIndex[R, G any] struct {
	projectionRegistration
	groups  KeyLayout[G]
	groupOf func(R) G
}

func (v *Vault) RegisterCountIndex[R, G any](
	bucket Name,
	version string,
	groups KeyLayout[G],
	groupOf func(R) G,
) (*CountIndex[R, G], error) {
	if err := v.provisionAmountsBucket(bucket); err != nil {
		return nil, err
	}

	return &CountIndex[R, G]{
		projectionRegistration: projectionRegistration{bucket: bucket, version: version},
		groups:                 groups,
		groupOf:                groupOf,
	}, nil
}

func (i *CountIndex[R, G]) Get(tx *Txn, group G) (int, error) {
	if err := i.refuseUnfollowed(); err != nil {
		return 0, err
	}
	count, err := tx.etx.Amounts(i.bucket).Get(i.groups.Encode(group).encoded)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", i.bucket, err)
	}

	return count, nil
}

func (i *CountIndex[R, G]) Scan(tx *Txn, groups KeyRange, fn func(G, int) (bool, error)) error {
	if err := i.refuseUnfollowed(); err != nil {
		return err
	}
	if err := tx.etx.Amounts(i.bucket).Scan(groups, func(key []byte, count int) (bool, error) {
		group, err := i.groups.Decode(key)
		if err != nil {
			return false, fmt.Errorf("decode %s group: %w", i.bucket, err)
		}

		return fn(group, count)
	}); err != nil {
		return fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return nil
}

func (i *CountIndex[R, G]) recordAdded(tx *Txn, record R) error {
	return i.raiseCount(tx, i.groupKeyOf(record), 1)
}

func (i *CountIndex[R, G]) groupKeyOf(record R) []byte {
	return i.groups.Encode(i.groupOf(record)).encoded
}

func (i *CountIndex[R, G]) raiseCount(tx *Txn, groupKey []byte, by uint) error {
	if err := tx.etx.Amounts(i.bucket).Raise(groupKey, by); err != nil {
		return fmt.Errorf("raise %s: %w", i.bucket, err)
	}

	return nil
}

func (i *CountIndex[R, G]) recordRemoved(tx *Txn, record R) error {
	return i.lowerCount(tx, i.groupKeyOf(record), 1)
}

func (i *CountIndex[R, G]) lowerCount(tx *Txn, groupKey []byte, by uint) error {
	if err := tx.etx.Amounts(i.bucket).Lower(groupKey, by); err != nil {
		return fmt.Errorf("lower %s: %w", i.bucket, err)
	}

	return nil
}

func (i *CountIndex[R, G]) recordReplaced(tx *Txn, previous, current R) error {
	previousKey, currentKey := i.groupKeyOf(previous), i.groupKeyOf(current)
	if bytes.Equal(previousKey, currentKey) {
		return nil
	}
	if err := i.lowerCount(tx, previousKey, 1); err != nil {
		return err
	}

	return i.raiseCount(tx, currentKey, 1)
}

func (i *CountIndex[R, G]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	counts, nextKey, err := i.countsFrom(tx, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for groupKey, count := range counts {
		if err := i.lowerCount(tx, []byte(groupKey), uint(count)); err != nil {
			return nil, err
		}
	}

	return nextKey, nil
}

func (i *CountIndex[R, G]) countsFrom(
	tx *Txn,
	firstKey []byte,
	limit int,
) (counts map[string]int, nextKey []byte, err error) {
	counts = make(map[string]int, limit)
	if err := tx.etx.Amounts(i.bucket).Scan(
		KeyRange{firstIncluded: firstKey},
		func(key []byte, count int) (bool, error) {
			if len(counts) == limit {
				nextKey = bytes.Clone(key)

				return false, nil
			}
			counts[string(key)] = count

			return true, nil
		},
	); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return counts, nextKey, nil
}
