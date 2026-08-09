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

func (layout TripleKey[A, B, C]) First(first A) Key {
	return keyOf(layout.first.items(first))
}

func (layout TripleKey[A, B, C]) FirstTwo(first A, second B) Key {
	items := layout.first.items(first)
	items = append(items, layout.second.items(second)...)

	return keyOf(items)
}

func (layout TripleKey[A, B, C]) Parts(key Key) (A, B, C, error) {
	firstTargets, firstValue := layout.first.holder()
	secondTargets, secondValue := layout.second.holder()
	thirdTargets, thirdValue := layout.third.holder()

	targets := firstTargets
	targets = append(targets, secondTargets...)
	targets = append(targets, thirdTargets...)

	if err := key.parseInto(targets); err != nil {
		var (
			unparsedFirst  A
			unparsedSecond B
			unparsedThird  C
		)

		return unparsedFirst, unparsedSecond, unparsedThird, err
	}

	return firstValue(), secondValue(), thirdValue(), nil
}
