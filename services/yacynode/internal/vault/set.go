package vault

type Set struct {
	entries *Collection[struct{}]
}

func RegisterSet(v *Vault, bucket Name) (*Set, error) {
	entries, err := Register(v, bucket, presenceCodec{})
	if err != nil {
		return nil, err
	}

	return &Set{entries: entries}, nil
}

func (s *Set) Add(tx *Txn, key Key) error {
	return s.entries.Put(tx, key, struct{}{})
}

func (s *Set) Remove(tx *Txn, key Key) (bool, error) {
	return s.entries.Delete(tx, key)
}

func (s *Set) Scan(tx *Txn, prefix Key, fn func(Key) (bool, error)) error {
	return s.entries.Scan(tx, prefix, func(key Key, _ struct{}) (bool, error) {
		return fn(key)
	})
}

func (s *Set) Len(tx *Txn) (int, error) {
	return s.entries.Len(tx)
}

type presenceCodec struct{}

func (presenceCodec) Encode(struct{}) ([]byte, error) { return []byte{}, nil }
func (presenceCodec) Decode([]byte) (struct{}, error) { return struct{}{}, nil }
