package vault

type SingleKeyLayout[A any] struct {
	first KeyPart[A]
}

func SingleKey[A any](first KeyPart[A]) SingleKeyLayout[A] {
	return SingleKeyLayout[A]{first: first}
}

func (layout SingleKeyLayout[A]) Key(first A) Key {
	return keyOf(layout.first.items(first))
}

func (layout SingleKeyLayout[A]) KeysWithFirst(first A) KeyRange {
	return layout.first.keysWith(first)
}

func (layout SingleKeyLayout[A]) KeysFromFirst(first A) KeyRange {
	return layout.first.keysFrom(first)
}

func (layout SingleKeyLayout[A]) KeysThroughFirst(first A) KeyRange {
	return layout.first.keysThrough(first)
}

func (layout SingleKeyLayout[A]) KeysBeforeFirst(first A) KeyRange {
	return layout.first.keysBefore(first)
}

func (layout SingleKeyLayout[A]) Parts(storedKey []byte) (A, error) {
	firstTargets, firstValue := layout.first.holder()

	if err := parseInto(storedKey, firstTargets); err != nil {
		var unparsed A

		return unparsed, err
	}

	return firstValue()
}
