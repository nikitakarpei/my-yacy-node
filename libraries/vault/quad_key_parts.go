package vault

type QuadKeyParts[A, B, C, D any] struct {
	first  KeyPart[A]
	second KeyPart[B]
	third  KeyPart[C]
	fourth KeyPart[D]
}

func QuadKey[A, B, C, D any](
	first KeyPart[A],
	second KeyPart[B],
	third KeyPart[C],
	fourth KeyPart[D],
) QuadKeyParts[A, B, C, D] {
	return QuadKeyParts[A, B, C, D]{first: first, second: second, third: third, fourth: fourth}
}

func (parts QuadKeyParts[A, B, C, D]) Key(first A, second B, third C, fourth D) Key {
	encoded := parts.first.appendTo(make([]byte, 0, keyCapacity), first)
	encoded = parts.second.appendTo(encoded, second)
	encoded = parts.third.appendTo(encoded, third)

	return Key{encoded: parts.fourth.appendTo(encoded, fourth)}
}

func (parts QuadKeyParts[A, B, C, D]) KeysWithFirst(first A) KeyRange {
	return parts.first.keysWith(first)
}

func (parts QuadKeyParts[A, B, C, D]) KeysBeforeFirst(first A) KeyRange {
	return parts.first.keysBefore(first)
}

func (parts QuadKeyParts[A, B, C, D]) KeysWithFirstThroughSecond(first A, second B) KeyRange {
	return parts.second.keysThrough(second).withPrefix(parts.first.encodingOf(first))
}

func (parts QuadKeyParts[A, B, C, D]) PartsOf(storedKey []byte) (A, B, C, D, error) {
	var (
		second B
		third  C
		fourth D
	)

	first, rest, err := parts.first.parseFrom(orderedBytes{encoded: storedKey})
	if err != nil {
		return first, second, third, fourth, err
	}
	second, rest, err = parts.second.parseFrom(rest)
	if err != nil {
		return first, second, third, fourth, err
	}
	third, rest, err = parts.third.parseFrom(rest)
	if err != nil {
		return first, second, third, fourth, err
	}
	fourth, rest, err = parts.fourth.parseFrom(rest)
	if err != nil {
		return first, second, third, fourth, err
	}

	return first, second, third, fourth, rest.leftoverError()
}

func (parts QuadKeyParts[A, B, C, D]) KeyLayoutFor[K any](
	partsOf func(K) (A, B, C, D),
	keyFrom func(A, B, C, D) K,
) KeyLayout[K] {
	return KeyLayout[K]{
		encode: func(key K) Key {
			return parts.Key(partsOf(key))
		},
		decode: func(storedKey []byte) (K, error) {
			first, second, third, fourth, err := parts.PartsOf(storedKey)
			if err != nil {
				var undecoded K

				return undecoded, err
			}

			return keyFrom(first, second, third, fourth), nil
		},
	}
}
