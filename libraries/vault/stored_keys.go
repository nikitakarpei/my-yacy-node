package vault

type storedKeys[K any] struct {
	identityOf func(K) string
	identities map[string]struct{}
}

func (s storedKeys[K]) holds(key K) bool {
	_, isHeld := s.identities[s.identityOf(key)]

	return isHeld
}

func (s storedKeys[K]) add(key K) {
	s.identities[s.identityOf(key)] = struct{}{}
}
