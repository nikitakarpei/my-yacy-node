package vault

type StoredBytes struct {
	stored []byte
}

func (b StoredBytes) AppendTo(buffer []byte) []byte {
	return append(buffer, b.stored...)
}
