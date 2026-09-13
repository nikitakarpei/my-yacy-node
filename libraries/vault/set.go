package vault

type Set[K any] struct {
	entries *Collection[K, struct{}]
}

func (v *Vault) RegisterSet[K any](bucket Name, keys KeyLayout[K]) (*Set[K], error) {
	entries, err := v.RegisterCollection(bucket, keys, presenceValueCodec{})
	if err != nil {
		return nil, err
	}

	return &Set[K]{entries: entries}, nil
}

func (s *Set[K]) Add(tx *Txn, key K) (alreadyExists bool, err error) {
	_, alreadyExists, err = s.entries.PutReturning(tx, key, struct{}{})
	if err != nil {
		return false, err
	}

	return alreadyExists, nil
}

func (s *Set[K]) Remove(tx *Txn, key K) (wasRemoved bool, err error) {
	wasRemoved, err = s.entries.Delete(tx, key)
	if err != nil {
		return false, err
	}

	return wasRemoved, nil
}

func (s *Set[K]) Scan(tx *Txn, keys KeyRange, fn func(K) (bool, error)) error {
	return s.entries.Scan(tx, keys, func(key K, _ struct{}) (bool, error) {
		return fn(key)
	})
}

func (s *Set[K]) Len(tx *Txn) (int, error) {
	return s.entries.Len(tx)
}

type presenceValueCodec struct{}

func (presenceValueCodec) Encode(struct{}) ([]byte, error) { return []byte{}, nil }
func (presenceValueCodec) Decode([]byte) (struct{}, error) { return struct{}{}, nil }
