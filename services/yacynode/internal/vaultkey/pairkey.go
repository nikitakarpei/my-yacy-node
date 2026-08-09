package vaultkey

type PairKey[A, B any] struct {
	first  Codec[A]
	second Codec[B]
}

func Pair[A, B any](first Codec[A], second Codec[B]) PairKey[A, B] {
	return PairKey[A, B]{first: first, second: second}
}

func (layout PairKey[A, B]) Key(first A, second B) Key {
	items := layout.first.items(first)
	items = append(items, layout.second.items(second)...)

	return keyOf(items)
}

func (layout PairKey[A, B]) First(first A) Key {
	return keyOf(layout.first.items(first))
}

func (layout PairKey[A, B]) Parts(key Key) (A, B, error) {
	firstTargets, firstValue := layout.first.holder()
	secondTargets, secondValue := layout.second.holder()

	if err := key.parseInto(append(firstTargets, secondTargets...)); err != nil {
		var (
			unparsedFirst  A
			unparsedSecond B
		)

		return unparsedFirst, unparsedSecond, err
	}

	return firstValue(), secondValue(), nil
}
