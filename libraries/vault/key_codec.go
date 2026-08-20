package vault

type KeyCodec[K any] interface {
	Encode(K) Key
	Decode([]byte) (K, error)
}
