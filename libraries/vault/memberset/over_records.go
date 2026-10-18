package memberset

import (
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func OverRecords(records vault.EngineRecords) vault.EngineMemberSets {
	return setsOverRecords{records: records}
}

type setsOverRecords struct {
	records vault.EngineRecords
}

func (s setsOverRecords) Members(key []byte) ([]vault.StoredMember, error) {
	encoded, err := s.records.Get(key)
	if err != nil || encoded == nil {
		return nil, err
	}

	return MembersFrom(encoded)
}

func (s setsOverRecords) Add(key []byte, member vault.StoredMember) error {
	members, err := s.Members(key)
	if err != nil || slices.Contains(members, member) {
		return err
	}

	return s.store(key, append(members, member))
}

func (s setsOverRecords) store(key []byte, members []vault.StoredMember) error {
	if _, err := s.records.Put(key, Change{Added: members}.Encoded()); err != nil {
		return fmt.Errorf("store member set: %w", err)
	}

	return nil
}

func (s setsOverRecords) Remove(key []byte, member vault.StoredMember) error {
	members, err := s.Members(key)
	if err != nil || (members != nil && !slices.Contains(members, member)) {
		return err
	}

	return s.store(key, slices.DeleteFunc(members, func(stored vault.StoredMember) bool {
		return stored == member
	}))
}

func (s setsOverRecords) Delete(key []byte) error {
	if _, err := s.records.Delete(key); err != nil {
		return fmt.Errorf("delete member set: %w", err)
	}

	return nil
}

func (s setsOverRecords) Scan(
	keys vault.KeyRange,
	fn func(key []byte) (bool, error),
) error {
	return s.records.Scan(keys, func(key, _ []byte) (bool, error) {
		return fn(key)
	})
}
