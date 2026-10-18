package vault

import (
	"fmt"
)

const secondPartIndexVersion = "1"

type SecondPartIndex[A, B, K any] struct {
	projectionRegistration
	pairKeyShape[A, B, K]
	swapped PairKeyParts[B, A]
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
		pairKeyShape: pairKeyShape[A, B, K]{parts: parts, partsOf: partsOf, keyFrom: keyFrom},
		swapped:      PairKey(parts.second, parts.first),
	}, nil
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

func (i *SecondPartIndex[A, B, K]) memberOf(key K) string {
	first, _ := i.partsOf(key)

	return string(i.parts.first.encodingOf(first))
}

func (i *SecondPartIndex[A, B, K]) storedMembersOf(
	tx *Txn,
	key K,
) (map[string]struct{}, error) {
	_, second := i.partsOf(key)
	storedMembers := make(map[string]struct{})
	err := i.visitKeysWith(tx, second, func(first A, _ B) (bool, error) {
		storedMembers[string(i.parts.first.encodingOf(first))] = struct{}{}

		return true, nil
	})

	return storedMembers, err
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
