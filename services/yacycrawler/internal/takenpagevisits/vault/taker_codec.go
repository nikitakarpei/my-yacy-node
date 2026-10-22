package vault

type takerCodec struct{}

func (takerCodec) Encode(taker string) ([]byte, error) { return []byte(taker), nil }

func (takerCodec) Decode(stored []byte) (string, error) { return string(stored), nil }
