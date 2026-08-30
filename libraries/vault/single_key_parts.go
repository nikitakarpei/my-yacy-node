package vault

type SingleKeyParts[A any] struct {
	first KeyPart[A]
}

func SingleKey[A any](first KeyPart[A]) SingleKeyParts[A] {
	return SingleKeyParts[A]{first: first}
}

func (parts SingleKeyParts[A]) Key(first A) Key {
	return keyOf(parts.first.items(first))
}

func (parts SingleKeyParts[A]) KeysWithFirst(first A) KeyRange {
	return parts.first.keysWith(first)
}

func (parts SingleKeyParts[A]) KeysFromFirst(first A) KeyRange {
	return parts.first.keysFrom(first)
}

func (parts SingleKeyParts[A]) KeysThroughFirst(first A) KeyRange {
	return parts.first.keysThrough(first)
}

func (parts SingleKeyParts[A]) KeysBeforeFirst(first A) KeyRange {
	return parts.first.keysBefore(first)
}

func (parts SingleKeyParts[A]) PartsOf(storedKey []byte) (A, error) {
	firstTargets, firstValue := parts.first.holder()

	if err := parseInto(storedKey, firstTargets); err != nil {
		var unparsed A

		return unparsed, err
	}

	return firstValue()
}

func (parts SingleKeyParts[A]) KeyLayout() KeyLayout[A] {
	return parts.KeyLayoutFor(
		func(first A) A { return first },
		func(first A) A { return first },
	)
}

func (parts SingleKeyParts[A]) KeyLayoutFor[K any](
	partsOf func(K) A,
	keyFrom func(A) K,
) KeyLayout[K] {
	return KeyLayout[K]{
		encode: func(key K) Key {
			return parts.Key(partsOf(key))
		},
		decode: func(storedKey []byte) (K, error) {
			first, err := parts.PartsOf(storedKey)
			if err != nil {
				var undecoded K

				return undecoded, err
			}

			return keyFrom(first), nil
		},
	}
}
