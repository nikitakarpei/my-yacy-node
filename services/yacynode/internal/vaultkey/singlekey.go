package vaultkey

type SingleKey[A any] struct {
	first Codec[A]
}

func Single[A any](first Codec[A]) SingleKey[A] {
	return SingleKey[A]{first: first}
}

func (layout SingleKey[A]) Key(first A) Key {
	return keyOf(layout.first.items(first))
}

func (layout SingleKey[A]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout SingleKey[A]) KeysFromFirst(first A) KeyRange {
	return layout.first.keysFrom(first)
}

func (layout SingleKey[A]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout SingleKey[A]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout SingleKey[A]) Parts(storedKey []byte) (A, error) {
	firstTargets, firstValue := layout.first.holder()

	if err := parseInto(storedKey, firstTargets); err != nil {
		var unparsed A

		return unparsed, err
	}

	return firstValue()
}
