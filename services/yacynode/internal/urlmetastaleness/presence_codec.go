package urlmetastaleness

// presenceCodec stores a url in the order bucket as a bare key, where the key
// itself is the whole fact and the value carries nothing.
type presenceCodec struct{}

func (presenceCodec) Encode(struct{}) ([]byte, error) { return []byte{}, nil }
func (presenceCodec) Decode([]byte) (struct{}, error) { return struct{}{}, nil }
