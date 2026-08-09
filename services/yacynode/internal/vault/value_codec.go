package vault

type ValueCodec[V any] interface {
	Encode(V) ([]byte, error)
	Decode([]byte) (V, error)
}
