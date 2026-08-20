package vault

type PairKeyLayout[A, B any] struct {
	first  KeyPart[A]
	second KeyPart[B]
}

func PairKey[A, B any](first KeyPart[A], second KeyPart[B]) PairKeyLayout[A, B] {
	return PairKeyLayout[A, B]{first: first, second: second}
}

func (layout PairKeyLayout[A, B]) Key(first A, second B) Key {
	items := layout.first.items(first)
	items = append(items, layout.second.items(second)...)

	return keyOf(items)
}

func (layout PairKeyLayout[A, B]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout PairKeyLayout[A, B]) KeysFromFirst(first A) KeyRange {
	return layout.first.keysFrom(first)
}

func (layout PairKeyLayout[A, B]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout PairKeyLayout[A, B]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout PairKeyLayout[A, B]) Parts(storedKey []byte) (A, B, error) {
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
