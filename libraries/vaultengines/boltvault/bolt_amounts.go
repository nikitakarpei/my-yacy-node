package boltvault

import (
	"encoding/binary"
	"fmt"

	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const amountWidth = 8

type boltAmounts struct {
	entries *bolt.Bucket
}

func (a boltAmounts) Get(key []byte) (int, error) {
	return amountFrom(a.entries.Get(key))
}

func amountFrom(raw []byte) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if len(raw) != amountWidth {
		return 0, fmt.Errorf("bad amount: %d bytes", len(raw))
	}

	//nolint:gosec // G115: a stored amount is always above zero and far below MaxInt64.
	return int(int64(binary.BigEndian.Uint64(raw))), nil
}

func (a boltAmounts) Raise(key []byte, by uint) error {
	return a.changeBy(key, int(by))
}

func (a boltAmounts) changeBy(key []byte, change int) error {
	current, err := a.Get(key)
	if err != nil {
		return err
	}

	changed := current + change
	if changed <= 0 {
		if err := a.entries.Delete(key); err != nil {
			return fmt.Errorf("delete amount: %w", err)
		}

		return nil
	}

	var raw [amountWidth]byte
	binary.BigEndian.PutUint64(raw[:], uint64(changed))
	if err := a.entries.Put(key, raw[:]); err != nil {
		return fmt.Errorf("store amount: %w", err)
	}

	return nil
}

func (a boltAmounts) Lower(key []byte, by uint) error {
	return a.changeBy(key, -int(by))
}

func (a boltAmounts) Scan(
	keys vault.KeyRange,
	fn func(key []byte, amount int) (bool, error),
) error {
	firstIncluded, firstExcluded := keys.Bounds()

	cursor := a.entries.Cursor()
	key, raw := firstEntryFrom(cursor, firstIncluded)
	for key != nil && isBeforeFirstExcluded(key, firstExcluded) {
		amount, err := amountFrom(raw)
		if err != nil {
			return err
		}
		keep, err := fn(key, amount)
		if err != nil || !keep {
			return err
		}

		key, raw = cursor.Next()
	}

	return nil
}
