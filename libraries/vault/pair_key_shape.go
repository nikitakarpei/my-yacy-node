package vault

type pairKeyShape[A, B, K any] struct {
	parts   PairKeyParts[A, B]
	partsOf func(K) (A, B)
	keyFrom func(A, B) K
}

func (s pairKeyShape[A, B, K]) keyLayout() KeyLayout[K] {
	return s.parts.KeyLayoutFor(s.partsOf, s.keyFrom)
}

func (s pairKeyShape[A, B, K]) groupOf(key K) string {
	_, second := s.partsOf(key)

	return string(s.groupKeyOf(second))
}

func (s pairKeyShape[A, B, K]) groupKeyOf(second B) []byte {
	return s.parts.second.encodingOf(second)
}
