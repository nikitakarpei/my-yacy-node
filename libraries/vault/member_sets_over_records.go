package vault

import (
	"fmt"
	"slices"
)

func MemberSetsOver(records EngineRecords) EngineMemberSets {
	return memberSetsOverRecords{records: records}
}

type memberSetsOverRecords struct {
	records EngineRecords
}

func (s memberSetsOverRecords) Members(key []byte) ([]StoredMember, error) {
	encoded, err := s.records.Get(key)
	if err != nil || encoded == nil {
		return nil, err
	}

	return StoredMembersFrom(encoded)
}

func (s memberSetsOverRecords) Add(key []byte, member StoredMember) error {
	members, err := s.Members(key)
	if err != nil || slices.Contains(members, member) {
		return err
	}

	return s.store(key, append(members, member))
}

func (s memberSetsOverRecords) store(key []byte, members []StoredMember) error {
	if _, err := s.records.Put(key, MemberSetChange{Added: members}.Encoded()); err != nil {
		return fmt.Errorf("store member set: %w", err)
	}

	return nil
}

func (s memberSetsOverRecords) Remove(key []byte, member StoredMember) error {
	members, err := s.Members(key)
	if err != nil || (members != nil && !slices.Contains(members, member)) {
		return err
	}

	return s.store(key, slices.DeleteFunc(members, func(stored StoredMember) bool {
		return stored == member
	}))
}

func (s memberSetsOverRecords) Delete(key []byte) error {
	if _, err := s.records.Delete(key); err != nil {
		return fmt.Errorf("delete member set: %w", err)
	}

	return nil
}

func (s memberSetsOverRecords) Scan(
	keys KeyRange,
	fn func(key []byte) (bool, error),
) error {
	return s.records.Scan(keys, func(key, _ []byte) (bool, error) {
		return fn(key)
	})
}
