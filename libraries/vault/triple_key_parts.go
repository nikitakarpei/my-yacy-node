package vault

type TripleKeyParts[A, B, C any] struct {
	first  KeyPart[A]
	second KeyPart[B]
	third  KeyPart[C]
}

func TripleKey[A, B, C any](
	first KeyPart[A],
	second KeyPart[B],
	third KeyPart[C],
) TripleKeyParts[A, B, C] {
	return TripleKeyParts[A, B, C]{first: first, second: second, third: third}
}

func (parts TripleKeyParts[A, B, C]) Key(first A, second B, third C) Key {
	encoded := parts.first.appendTo(make([]byte, 0, keyCapacity), first)
	encoded = parts.second.appendTo(encoded, second)

	return Key{encoded: parts.third.appendTo(encoded, third)}
}

func (parts TripleKeyParts[A, B, C]) KeysWithFirst(first A) KeyRange {
	return parts.first.keysWith(first)
}

func (parts TripleKeyParts[A, B, C]) KeysFromFirst(first A) KeyRange {
	return parts.first.keysFrom(first)
}

func (parts TripleKeyParts[A, B, C]) KeysThroughFirst(first A) KeyRange {
	return parts.first.keysThrough(first)
}

func (parts TripleKeyParts[A, B, C]) KeysBeforeFirst(first A) KeyRange {
	return parts.first.keysBefore(first)
}

func (parts TripleKeyParts[A, B, C]) PartsOf(storedKey []byte) (A, B, C, error) {
	var (
		second B
		third  C
	)

	first, rest, err := parts.first.parseFrom(orderedBytes{encoded: storedKey})
	if err != nil {
		return first, second, third, err
	}
	second, rest, err = parts.second.parseFrom(rest)
	if err != nil {
		return first, second, third, err
	}
	third, rest, err = parts.third.parseFrom(rest)
	if err != nil {
		return first, second, third, err
	}

	return first, second, third, rest.leftoverError()
}

func (parts TripleKeyParts[A, B, C]) KeyLayoutFor[K any](
	partsOf func(K) (A, B, C),
	keyFrom func(A, B, C) K,
) KeyLayout[K] {
	return KeyLayout[K]{
		encode: func(key K) Key {
			return parts.Key(partsOf(key))
		},
		decode: func(storedKey []byte) (K, error) {
			first, second, third, err := parts.PartsOf(storedKey)
			if err != nil {
				var undecoded K

				return undecoded, err
			}

			return keyFrom(first, second, third), nil
		},
	}
}
