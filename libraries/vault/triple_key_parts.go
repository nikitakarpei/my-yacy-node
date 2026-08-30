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
	items := parts.first.items(first)
	items = append(items, parts.second.items(second)...)
	items = append(items, parts.third.items(third)...)

	return keyOf(items)
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
	firstTargets, firstValue := parts.first.holder()
	secondTargets, secondValue := parts.second.holder()
	thirdTargets, thirdValue := parts.third.holder()

	targets := firstTargets
	targets = append(targets, secondTargets...)
	targets = append(targets, thirdTargets...)

	var (
		first  A
		second B
		third  C
	)

	if err := parseInto(storedKey, targets); err != nil {
		return first, second, third, err
	}

	first, err := firstValue()
	if err != nil {
		return first, second, third, err
	}

	second, err = secondValue()
	if err != nil {
		return first, second, third, err
	}

	third, err = thirdValue()
	if err != nil {
		return first, second, third, err
	}

	return first, second, third, nil
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
