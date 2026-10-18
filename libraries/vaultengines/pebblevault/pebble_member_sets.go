package pebblevault

import "github.com/nikitakarpei/yacy-rwi-node/vault"

type pebbleMemberSets struct {
	entries storedEntries
	changes *memberSetChanges
}

func (s pebbleMemberSets) Members(key []byte) ([]vault.StoredMember, error) {
	absoluteKey := s.entries.absoluteKeyFrom(key)
	storedMembers, err := s.storedMembersAt(absoluteKey)
	if err != nil {
		return nil, err
	}

	return s.changes.appliedTo(absoluteKey, storedMembers), nil
}

func (s pebbleMemberSets) storedMembersAt(absoluteKey []byte) ([]vault.StoredMember, error) {
	encoded, err := s.entries.valueAtAbsoluteKey(absoluteKey)
	if err != nil || encoded == nil {
		return nil, err
	}

	return vault.StoredMembersFrom(encoded)
}

func (s pebbleMemberSets) Add(key []byte, member vault.StoredMember) error {
	s.changes.mark(s.entries.absoluteKeyFrom(key), member, true)

	return nil
}

func (s pebbleMemberSets) Remove(key []byte, member vault.StoredMember) error {
	s.changes.mark(s.entries.absoluteKeyFrom(key), member, false)

	return nil
}

func (s pebbleMemberSets) Delete(key []byte) error {
	absoluteKey := s.entries.absoluteKeyFrom(key)
	s.changes.drop(absoluteKey)

	return s.entries.removeAtAbsoluteKey(absoluteKey)
}

func (s pebbleMemberSets) Scan(
	keys vault.KeyRange,
	fn func(key []byte) (bool, error),
) error {
	if err := s.changes.stageWithin(s.entries.boundsFor(keys)); err != nil {
		return err
	}

	return s.entries.visit(keys, func(key, _ []byte) (bool, error) {
		return fn(key)
	})
}
