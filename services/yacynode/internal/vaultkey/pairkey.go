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

func (layout PairKey[A, B]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout PairKey[A, B]) KeysFromFirst(first A) KeyRange {
	return layout.first.keysFrom(first)
}

func (layout PairKey[A, B]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout PairKey[A, B]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout PairKey[A, B]) Parts(storedKey []byte) (A, B, error) {
	firstTargets, firstValue := layout.first.holder()
	secondTargets, secondValue := layout.second.holder()

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
