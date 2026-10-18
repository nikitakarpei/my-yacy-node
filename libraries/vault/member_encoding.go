package vault

type MemberEncoding[A any] struct {
	dictionary *FixedDictionary[A]
}

func MemberEncodingFrom[A any](dictionary *FixedDictionary[A]) MemberEncoding[A] {
	return MemberEncoding[A]{dictionary: dictionary}
}

func (e MemberEncoding[A]) versionFrom(version string) string {
	return version + ":" + e.dictionary.fingerprint
}

func (e MemberEncoding[A]) storedMemberFrom(value A) StoredMember {
	if id, found := e.dictionary.idOf(value); found {
		return StoredMember{ID: id}
	}

	return StoredMember{Raw: string(e.dictionary.encodingOf(value))}
}

func (e MemberEncoding[A]) valueFrom(member StoredMember) (A, error) {
	if member.Form() == MemberFormRaw {
		return e.dictionary.valueFrom([]byte(member.Raw))
	}

	return e.dictionary.valueAt(member.ID)
}
