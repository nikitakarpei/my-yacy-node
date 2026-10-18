package vault

type memberSetLayout[A, B any] struct {
	memberSetKeys KeyPart[B]
	members       MemberEncoding[A]
}

type decodedMembers[A any] struct {
	members           []A
	decodableMembers  []StoredMember
	undecodableAmount int
	undecodableCause  error
}

func (l memberSetLayout[A, B]) memberSetKeyOf(second B) []byte {
	return l.memberSetKeys.encodingOf(second)
}

func (l memberSetLayout[A, B]) storedMemberFrom(first A) StoredMember {
	return l.members.storedMemberFrom(first)
}

func (l memberSetLayout[A, B]) decodedFrom(storedMembers []StoredMember) decodedMembers[A] {
	decoded := decodedMembers[A]{
		members:          make([]A, 0, len(storedMembers)),
		decodableMembers: make([]StoredMember, 0, len(storedMembers)),
	}
	for _, storedMember := range storedMembers {
		member, err := l.members.valueFrom(storedMember)
		if err != nil {
			decoded.undecodableAmount++
			decoded.undecodableCause = err

			continue
		}
		decoded.members = append(decoded.members, member)
		decoded.decodableMembers = append(decoded.decodableMembers, storedMember)
	}

	return decoded
}
