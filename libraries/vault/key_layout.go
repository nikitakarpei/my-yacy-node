package vault

type KeyLayout[K any] struct {
	encode func(K) Key
	decode func([]byte) (K, error)
}

func (layout KeyLayout[K]) Encode(key K) Key {
	return layout.encode(key)
}

func (layout KeyLayout[K]) Decode(storedKey []byte) (K, error) {
	return layout.decode(storedKey)
}
