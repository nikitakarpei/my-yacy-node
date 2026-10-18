package vault

import (
	"bytes"
	"fmt"
)

type MemberSetIndex[A, B, K any] struct {
	projectionRegistration
	pairKeyShape[A, B, K]
	memberSetKeys KeyPart[B]
	members       MemberEncoding[A]
	observer      MemberSetObserver
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func (v *Vault) RegisterMemberSetIndex[A, B, K any](
	bucket Name,
	version string,
	parts PairKeyParts[A, B],
	partsOf func(K) (A, B),
	keyFrom func(A, B) K,
	members MemberEncoding[A],
	observer MemberSetObserver,
) (*MemberSetIndex[A, B, K], error) {
	if err := v.provisionMemberSetsBucket(bucket); err != nil {
		return nil, err
	}

	return &MemberSetIndex[A, B, K]{
		projectionRegistration: projectionRegistration{
			bucket:  bucket,
			version: members.versionFrom(version),
		},
		pairKeyShape:  pairKeyShape[A, B, K]{parts: parts, partsOf: partsOf, keyFrom: keyFrom},
		memberSetKeys: parts.second,
		members:       members,
		observer:      memberSetObserverOrSilent(observer),
	}, nil
}

func (i *MemberSetIndex[A, B, K]) MembersOf(tx *Txn, second B) ([]A, error) {
	if err := i.refuseUnfollowed(); err != nil {
		return nil, err
	}
	memberSetKey := i.memberSetKeyOf(second)
	storedMembers, err := i.storedMembersAt(tx, memberSetKey)
	if err != nil {
		return nil, err
	}
	members, decodableMembers := i.decodedFrom(storedMembers)
	readSets := i.readSetsIn(tx)
	readSets.membersRead(memberSetKey, decodableMembers)

	return members, nil
}

func (i *MemberSetIndex[A, B, K]) readSetsIn(tx *Txn) *readMemberSets {
	return tx.pendingWorkFor(i, func() *readMemberSets { return newReadMemberSets(i.bucket) })
}

func (i *MemberSetIndex[A, B, K]) storedMembersAt(
	tx *Txn,
	memberSetKey []byte,
) ([]StoredMember, error) {
	storedMembers, err := tx.etx.MemberSets(i.bucket).Members(memberSetKey)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", i.bucket, err)
	}

	return storedMembers, nil
}

func (i *MemberSetIndex[A, B, K]) decodedFrom(
	storedMembers []StoredMember,
) (members []A, decodableMembers []StoredMember) {
	members = make([]A, 0, len(storedMembers))
	decodableMembers = make([]StoredMember, 0, len(storedMembers))
	var undecodable int
	var cause error
	for _, storedMember := range storedMembers {
		member, err := i.members.valueFrom(storedMember)
		if err != nil {
			undecodable++
			cause = err

			continue
		}
		members = append(members, member)
		decodableMembers = append(decodableMembers, storedMember)
	}
	if undecodable > 0 {
		reportUndecodableMembers(i.observer, i.bucket, undecodable, cause)
	}

	return members, decodableMembers
}

func (i *MemberSetIndex[A, B, K]) keyAdded(tx *Txn, key K) error {
	first, second := i.partsOf(key)

	return i.addMember(tx, i.memberSetKeyOf(second), i.members.storedMemberFrom(first))
}

func (i *MemberSetIndex[A, B, K]) memberSetKeyOf(second B) []byte {
	return i.memberSetKeys.encodingOf(second)
}

func (i *MemberSetIndex[A, B, K]) addMember(
	tx *Txn,
	memberSetKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Add(memberSetKey, storedMember); err != nil {
		return fmt.Errorf("add to %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberAdded(memberSetKey)
	i.observer.ObserveMemberAdded(i.bucket, storedMember.Form())

	return nil
}

func (i *MemberSetIndex[A, B, K]) keyRemoved(tx *Txn, key K) error {
	first, second := i.partsOf(key)

	return i.removeMember(tx, i.memberSetKeyOf(second), i.members.storedMemberFrom(first))
}

func (i *MemberSetIndex[A, B, K]) removeMember(
	tx *Txn,
	memberSetKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Remove(memberSetKey, storedMember); err != nil {
		return fmt.Errorf("remove from %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberRemoved(memberSetKey, storedMember)

	return nil
}

func (i *MemberSetIndex[A, B, K]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	storedMemberSetKeys, nextKey, err := i.storedMemberSetKeysFrom(tx, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for _, memberSetKey := range storedMemberSetKeys {
		if err := tx.etx.MemberSets(i.bucket).Delete(memberSetKey); err != nil {
			return nil, fmt.Errorf("delete %s set: %w", i.bucket, err)
		}
	}

	return nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) storedMemberSetKeysFrom(
	tx *Txn,
	firstKey []byte,
	limit int,
) (storedMemberSetKeys [][]byte, nextKey []byte, err error) {
	if err := tx.etx.MemberSets(i.bucket).Scan(
		KeyRange{firstIncluded: firstKey},
		func(memberSetKey []byte) (bool, error) {
			if len(storedMemberSetKeys) == limit {
				nextKey = bytes.Clone(memberSetKey)

				return false, nil
			}
			storedMemberSetKeys = append(storedMemberSetKeys, bytes.Clone(memberSetKey))

			return true, nil
		},
	); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return storedMemberSetKeys, nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) storedKeysWithSecondPartOf(
	tx *Txn,
	key K,
) (storedKeys[K], error) {
	_, second := i.partsOf(key)
	storedMembers, err := i.storedMembersAt(tx, i.memberSetKeyOf(second))
	if err != nil {
		return storedKeys[K]{}, err
	}
	keys := storedKeys[K]{
		identityOf: i.identityOf,
		identities: make(map[string]struct{}, len(storedMembers)),
	}
	for _, storedMember := range storedMembers {
		keys.identities[storedMember.identity()] = struct{}{}
	}

	return keys, nil
}

func (i *MemberSetIndex[A, B, K]) identityOf(key K) string {
	first, _ := i.partsOf(key)

	return i.members.storedMemberFrom(first).identity()
}
