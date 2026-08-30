package vault

type TripleKeyLayout[A, B, C any] struct {
	first  KeyPart[A]
	second KeyPart[B]
	third  KeyPart[C]
}

func TripleKey[A, B, C any](
	first KeyPart[A],
	second KeyPart[B],
	third KeyPart[C],
) TripleKeyLayout[A, B, C] {
	return TripleKeyLayout[A, B, C]{first: first, second: second, third: third}
}

func (layout TripleKeyLayout[A, B, C]) Key(first A, second B, third C) Key {
	items := layout.first.items(first)
	items = append(items, layout.second.items(second)...)
	items = append(items, layout.third.items(third)...)

	return keyOf(items)
}

func (layout TripleKeyLayout[A, B, C]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout TripleKeyLayout[A, B, C]) KeysFromFirst(first A) KeyRange {
	return layout.first.keysFrom(first)
}

func (layout TripleKeyLayout[A, B, C]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout TripleKeyLayout[A, B, C]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout TripleKeyLayout[A, B, C]) Parts(storedKey []byte) (A, B, C, error) {
	firstTargets, firstValue := layout.first.holder()
	secondTargets, secondValue := layout.second.holder()
	thirdTargets, thirdValue := layout.third.holder()

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

func (layout TripleKeyLayout[A, B, C]) KeyCodecFor[K any](
	partsOf func(K) (A, B, C),
	keyFrom func(A, B, C) K,
) KeyCodec[K] {
	return keyCodec[K]{
		encode: func(key K) Key {
			return layout.Key(partsOf(key))
		},
		decode: func(storedKey []byte) (K, error) {
			first, second, third, err := layout.Parts(storedKey)
			if err != nil {
				var undecoded K

				return undecoded, err
			}

			return keyFrom(first, second, third), nil
		},
	}
}
