package vault

type KeyCodec[K any] interface {
	Encode(K) Key
	Decode([]byte) (K, error)
}

type keyCodec[K any] struct {
	encode func(K) Key
	decode func([]byte) (K, error)
}

func (codec keyCodec[K]) Encode(key K) Key {
	return codec.encode(key)
}

func (codec keyCodec[K]) Decode(storedKey []byte) (K, error) {
	return codec.decode(storedKey)
}
