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

func (layout SingleKey[A]) Parts(key Key) (A, error) {
	firstTargets, firstValue := layout.first.holder()

	if err := key.parseInto(firstTargets); err != nil {
		var unparsed A

		return unparsed, err
	}

	return firstValue(), nil
}
