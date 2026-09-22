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
	items := parts.first.items(first)
	items = append(items, parts.second.items(second)...)
	items = append(items, parts.third.items(third)...)
	items = append(items, parts.fourth.items(fourth)...)

	return keyOf(items)
}

func (parts QuadKeyParts[A, B, C, D]) KeysWithFirst(first A) KeyRange {
	return parts.first.keysWith(first)
}

func (parts QuadKeyParts[A, B, C, D]) KeysWithFirstThroughSecond(first A, second B) KeyRange {
	return parts.second.keysThrough(second).withPrefix(parts.first.encodingOf(first))
}

func (parts QuadKeyParts[A, B, C, D]) PartsOf(storedKey []byte) (A, B, C, D, error) {
	firstTargets, firstValue := parts.first.holder()
	secondTargets, secondValue := parts.second.holder()
	thirdTargets, thirdValue := parts.third.holder()
	fourthTargets, fourthValue := parts.fourth.holder()

	targets := firstTargets
	targets = append(targets, secondTargets...)
	targets = append(targets, thirdTargets...)
	targets = append(targets, fourthTargets...)

	var (
		first  A
		second B
		third  C
		fourth D
	)

	if err := parseInto(storedKey, targets); err != nil {
		return first, second, third, fourth, err
	}

	first, err := firstValue()
	if err != nil {
		return first, second, third, fourth, err
	}

	second, err = secondValue()
	if err != nil {
		return first, second, third, fourth, err
	}

	third, err = thirdValue()
	if err != nil {
		return first, second, third, fourth, err
	}

	fourth, err = fourthValue()
	if err != nil {
		return first, second, third, fourth, err
	}

	return first, second, third, fourth, nil
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
