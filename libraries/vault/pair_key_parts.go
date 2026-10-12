package vault

type PairKeyParts[A, B any] struct {
	first  KeyPart[A]
	second KeyPart[B]
}

func PairKey[A, B any](first KeyPart[A], second KeyPart[B]) PairKeyParts[A, B] {
	return PairKeyParts[A, B]{first: first, second: second}
}

func (parts PairKeyParts[A, B]) Key(first A, second B) Key {
	encoded := parts.first.appendTo(make([]byte, 0, keyCapacity), first)

	return Key{encoded: parts.second.appendTo(encoded, second)}
}

func (parts PairKeyParts[A, B]) KeysWithFirst(first A) KeyRange {
	return parts.first.keysWith(first)
}

func (parts PairKeyParts[A, B]) KeysFromFirst(first A) KeyRange {
	return parts.first.keysFrom(first)
}

func (parts PairKeyParts[A, B]) KeysThroughFirst(first A) KeyRange {
	return parts.first.keysThrough(first)
}

func (parts PairKeyParts[A, B]) KeysBeforeFirst(first A) KeyRange {
	return parts.first.keysBefore(first)
}

func (parts PairKeyParts[A, B]) KeysWithFirstBetweenSeconds(first A, lowest, highest B) KeyRange {
	return parts.second.keysBetween(lowest, highest).withPrefix(parts.first.encodingOf(first))
}

func (parts PairKeyParts[A, B]) PartsOf(storedKey []byte) (A, B, error) {
	var second B

	first, rest, err := parts.first.parseFrom(orderedBytes{encoded: storedKey})
	if err != nil {
		return first, second, err
	}
	second, rest, err = parts.second.parseFrom(rest)
	if err != nil {
		return first, second, err
	}

	return first, second, rest.leftoverError()
}

func (parts PairKeyParts[A, B]) KeyLayoutFor[K any](
	partsOf func(K) (A, B),
	keyFrom func(A, B) K,
) KeyLayout[K] {
	return KeyLayout[K]{
		encode: func(key K) Key {
			return parts.Key(partsOf(key))
		},
		decode: func(storedKey []byte) (K, error) {
			first, second, err := parts.PartsOf(storedKey)
			if err != nil {
				var undecoded K

				return undecoded, err
			}

			return keyFrom(first, second), nil
		},
	}
}
