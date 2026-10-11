package vault

type PairKeyParts[A, B any] struct {
	first  KeyPart[A]
	second KeyPart[B]
}

func PairKey[A, B any](first KeyPart[A], second KeyPart[B]) PairKeyParts[A, B] {
	return PairKeyParts[A, B]{first: first, second: second}
}

func (parts PairKeyParts[A, B]) Key(first A, second B) Key {
	items := parts.first.items(first)
	items = append(items, parts.second.items(second)...)

	return keyOf(items)
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

func (parts PairKeyParts[A, B]) PartsOf(storedKey []byte) (A, B, error) {
	firstTargets, firstValue := parts.first.holder()
	secondTargets, secondValue := parts.second.holder()

	var (
		first  A
		second B
	)

	if err := parseInto(storedKey, append(firstTargets, secondTargets...)); err != nil {
		return first, second, err
	}

	first, err := firstValue()
	if err != nil {
		return first, second, err
	}

	second, err = secondValue()
	if err != nil {
		return first, second, err
	}

	return first, second, nil
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
