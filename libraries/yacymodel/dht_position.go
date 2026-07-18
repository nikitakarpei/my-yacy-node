package yacymodel

const MaxPosition = uint64(1)<<63 - 1

// TODO: DHT position is derived by reading the hash's base64 text, coupling
// routing to the transport alphabet; blocked with the alphabet itself.
func Position(h Hash) (uint64, error) {
	return cardinal(string(h))
}

func Distance(from, to uint64) uint64 {
	if to >= from {
		return to - from
	}
	return (MaxPosition - from) + to + 1
}
