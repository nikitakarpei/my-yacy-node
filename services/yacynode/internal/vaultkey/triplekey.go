package vaultkey

type TripleKey[A, B, C any] struct {
	first  Codec[A]
	second Codec[B]
	third  Codec[C]
}

func Triple[A, B, C any](first Codec[A], second Codec[B], third Codec[C]) TripleKey[A, B, C] {
	return TripleKey[A, B, C]{first: first, second: second, third: third}
}

func (layout TripleKey[A, B, C]) Key(first A, second B, third C) Key {
	items := layout.first.items(first)
	items = append(items, layout.second.items(second)...)
	items = append(items, layout.third.items(third)...)

	return keyOf(items)
}

func (layout TripleKey[A, B, C]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout TripleKey[A, B, C]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout TripleKey[A, B, C]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout TripleKey[A, B, C]) Parts(key Key) (A, B, C, error) {
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

	if err := key.parseInto(targets); err != nil {
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
