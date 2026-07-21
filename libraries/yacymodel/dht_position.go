package yacymodel

const MaxPosition = uint64(1)<<63 - 1

func Position(h Hash) (uint64, error) {
	return cardinal(h.String())
}

func Distance(from, to uint64) uint64 {
	if to >= from {
		return to - from
	}
	return (MaxPosition - from) + to + 1
}
