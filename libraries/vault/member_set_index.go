package vault

import (
	"bytes"
	"fmt"
)

type MemberSetIndex[A, B, K any] struct {
	projectionRegistration
	pairKeyShape[A, B, K]
	dictionary *FixedDictionary[A]
	observer   MemberSetObserver
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func (v *Vault) RegisterMemberSetIndex[A, B, K any](
	bucket Name,
	version string,
	parts PairKeyParts[A, B],
	partsOf func(K) (A, B),
	keyFrom func(A, B) K,
	dictionary *FixedDictionary[A],
	observer MemberSetObserver,
) (*MemberSetIndex[A, B, K], error) {
	if err := v.provisionMemberSetsBucket(bucket); err != nil {
		return nil, err
	}

	return &MemberSetIndex[A, B, K]{
		projectionRegistration: projectionRegistration{
			bucket:  bucket,
			version: version + ":" + dictionary.fingerprint,
		},
		pairKeyShape: pairKeyShape[A, B, K]{parts: parts, partsOf: partsOf, keyFrom: keyFrom},
		dictionary:   dictionary,
		observer:     memberSetObserverOrSilent(observer),
	}, nil
}

func (i *MemberSetIndex[A, B, K]) MembersOf(tx *Txn, second B) ([]A, error) {
	if err := i.refuseUnfollowed(); err != nil {
		return nil, err
	}
	secondPartKey := i.secondPartKeyOf(second)
	storedMembers, err := i.storedMembersAt(tx, secondPartKey)
	if err != nil {
		return nil, err
	}
	members, decodableMembers := i.decodedFrom(storedMembers)
	readSets := i.readSetsIn(tx)
	readSets.membersRead(secondPartKey, decodableMembers)

	return members, nil
}

func (i *MemberSetIndex[A, B, K]) readSetsIn(tx *Txn) *readMemberSets {
	return tx.pendingWorkFor(i, func() *readMemberSets { return newReadMemberSets(i.bucket) })
}

func (i *MemberSetIndex[A, B, K]) storedMembersAt(
	tx *Txn,
	secondPartKey []byte,
) ([]StoredMember, error) {
	storedMembers, err := tx.etx.MemberSets(i.bucket).Members(secondPartKey)
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
		member, err := i.dictionary.valueOf(storedMember)
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
	secondPartKey, storedMember := i.storedPartsOf(key)

	return i.addMember(tx, secondPartKey, storedMember)
}

func (i *MemberSetIndex[A, B, K]) storedPartsOf(key K) ([]byte, StoredMember) {
	first, second := i.partsOf(key)

	return i.secondPartKeyOf(second), i.dictionary.storedMemberOf(first)
}

func (i *MemberSetIndex[A, B, K]) addMember(
	tx *Txn,
	secondPartKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Add(secondPartKey, storedMember); err != nil {
		return fmt.Errorf("add to %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberAdded(secondPartKey)
	i.observer.ObserveMemberAdded(i.bucket, storedMember.form())

	return nil
}

func (i *MemberSetIndex[A, B, K]) keyRemoved(tx *Txn, key K) error {
	secondPartKey, storedMember := i.storedPartsOf(key)

	return i.removeMember(tx, secondPartKey, storedMember)
}

func (i *MemberSetIndex[A, B, K]) removeMember(
	tx *Txn,
	secondPartKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Remove(secondPartKey, storedMember); err != nil {
		return fmt.Errorf("remove from %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberRemoved(secondPartKey, storedMember)

	return nil
}

func (i *MemberSetIndex[A, B, K]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	secondPartKeys, nextKey, err := i.secondPartKeysFrom(tx, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for _, secondPartKey := range secondPartKeys {
		if err := tx.etx.MemberSets(i.bucket).Delete(secondPartKey); err != nil {
			return nil, fmt.Errorf("delete %s set: %w", i.bucket, err)
		}
	}

	return nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) secondPartKeysFrom(
	tx *Txn,
	firstKey []byte,
	limit int,
) (secondPartKeys [][]byte, nextKey []byte, err error) {
	if err := tx.etx.MemberSets(i.bucket).Scan(
		KeyRange{firstIncluded: firstKey},
		func(secondPartKey []byte) (bool, error) {
			if len(secondPartKeys) == limit {
				nextKey = bytes.Clone(secondPartKey)

				return false, nil
			}
			secondPartKeys = append(secondPartKeys, bytes.Clone(secondPartKey))

			return true, nil
		},
	); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return secondPartKeys, nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) storedKeysWithSecondPartOf(
	tx *Txn,
	key K,
) (storedKeys[K], error) {
	secondPartKey, _ := i.storedPartsOf(key)
	storedMembers, err := i.storedMembersAt(tx, secondPartKey)
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
	_, storedMember := i.storedPartsOf(key)

	return storedMember.identity()
}
