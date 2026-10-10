package vault

import "fmt"

type Amounts[K any] struct {
	name Name
	keys KeyLayout[K]
}

func (v *Vault) RegisterAmounts[K any](bucket Name, keys KeyLayout[K]) (*Amounts[K], error) {
	if err := v.provisionAmountsBucket(bucket); err != nil {
		return nil, err
	}

	return &Amounts[K]{name: bucket, keys: keys}, nil
}

func (a *Amounts[K]) Get(tx *Txn, key K) (int, error) {
	amount, err := tx.etx.Amounts(a.name).Get(a.keys.Encode(key).Bytes())
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", a.name, err)
	}

	return amount, nil
}

func (a *Amounts[K]) Raise(tx *Txn, key K, by uint) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	if err := tx.etx.Amounts(a.name).Raise(a.keys.Encode(key).Bytes(), by); err != nil {
		return fmt.Errorf("raise %s: %w", a.name, err)
	}

	return nil
}

func (a *Amounts[K]) Lower(tx *Txn, key K, by uint) error {
	if !tx.etx.Writable() {
		return errReadOnly
	}
	tx.calledWriteOperation = true

	if err := tx.etx.Amounts(a.name).Lower(a.keys.Encode(key).Bytes(), by); err != nil {
		return fmt.Errorf("lower %s: %w", a.name, err)
	}

	return nil
}

func (a *Amounts[K]) Scan(tx *Txn, keys KeyRange, fn func(K, int) (bool, error)) error {
	if err := tx.etx.Amounts(a.name).Scan(keys, func(key []byte, amount int) (bool, error) {
		decodedKey, err := a.keys.Decode(key)
		if err != nil {
			return false, fmt.Errorf("decode %s key: %w", a.name, err)
		}

		return fn(decodedKey, amount)
	}); err != nil {
		return fmt.Errorf("scan %s: %w", a.name, err)
	}

	return nil
}
