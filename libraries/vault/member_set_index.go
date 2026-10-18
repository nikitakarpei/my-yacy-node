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

func (i *MemberSetIndex[A, B, K]) MembersOf(tx *Txn, group B) ([]A, error) {
	if err := i.refuseUnfollowed(); err != nil {
		return nil, err
	}
	groupKey := i.groupKeyOf(group)
	storedMembers, err := i.storedMembersAt(tx, groupKey)
	if err != nil {
		return nil, err
	}
	members, decodableMembers := i.decodedFrom(storedMembers)
	readSets := i.readSetsIn(tx)
	readSets.membersRead(groupKey, decodableMembers)

	return members, nil
}

func (i *MemberSetIndex[A, B, K]) readSetsIn(tx *Txn) *readMemberSets {
	return tx.pendingWorkFor(i, func() *readMemberSets { return newReadMemberSets(i.bucket) })
}

func (i *MemberSetIndex[A, B, K]) storedMembersAt(
	tx *Txn,
	groupKey []byte,
) ([]StoredMember, error) {
	storedMembers, err := tx.etx.MemberSets(i.bucket).Members(groupKey)
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
	groupKey, storedMember := i.storedPartsOf(key)

	return i.addMember(tx, groupKey, storedMember)
}

func (i *MemberSetIndex[A, B, K]) storedPartsOf(key K) ([]byte, StoredMember) {
	member, group := i.partsOf(key)

	return i.groupKeyOf(group), i.dictionary.storedMemberOf(member)
}

func (i *MemberSetIndex[A, B, K]) addMember(
	tx *Txn,
	groupKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Add(groupKey, storedMember); err != nil {
		return fmt.Errorf("add to %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberAdded(groupKey)
	i.observer.ObserveMemberAdded(i.bucket, storedMember.form())

	return nil
}

func (i *MemberSetIndex[A, B, K]) keyRemoved(tx *Txn, key K) error {
	groupKey, storedMember := i.storedPartsOf(key)

	return i.removeMember(tx, groupKey, storedMember)
}

func (i *MemberSetIndex[A, B, K]) removeMember(
	tx *Txn,
	groupKey []byte,
	storedMember StoredMember,
) error {
	if err := tx.etx.MemberSets(i.bucket).Remove(groupKey, storedMember); err != nil {
		return fmt.Errorf("remove from %s: %w", i.bucket, err)
	}
	readSets := i.readSetsIn(tx)
	readSets.memberRemoved(groupKey, storedMember)

	return nil
}

func (i *MemberSetIndex[A, B, K]) clearFrom(tx *Txn, firstKey []byte, limit int) ([]byte, error) {
	groupKeys, nextKey, err := i.groupKeysFrom(tx, firstKey, limit)
	if err != nil {
		return nil, err
	}
	for _, groupKey := range groupKeys {
		if err := tx.etx.MemberSets(i.bucket).Delete(groupKey); err != nil {
			return nil, fmt.Errorf("delete %s set: %w", i.bucket, err)
		}
	}

	return nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) groupKeysFrom(
	tx *Txn,
	firstKey []byte,
	limit int,
) (groupKeys [][]byte, nextKey []byte, err error) {
	if err := tx.etx.MemberSets(i.bucket).Scan(
		KeyRange{firstIncluded: firstKey},
		func(groupKey []byte, _ []StoredMember) (bool, error) {
			if len(groupKeys) == limit {
				nextKey = bytes.Clone(groupKey)

				return false, nil
			}
			groupKeys = append(groupKeys, bytes.Clone(groupKey))

			return true, nil
		},
	); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", i.bucket, err)
	}

	return groupKeys, nextKey, nil
}

func (i *MemberSetIndex[A, B, K]) memberIdentityOf(key K) string {
	_, storedMember := i.storedPartsOf(key)

	return storedMember.identity()
}

func (i *MemberSetIndex[A, B, K]) storedIdentitiesOf(
	tx *Txn,
	key K,
) (map[string]struct{}, error) {
	groupKey, _ := i.storedPartsOf(key)
	storedMembers, err := i.storedMembersAt(tx, groupKey)
	if err != nil {
		return nil, err
	}
	storedIdentities := make(map[string]struct{}, len(storedMembers))
	for _, storedMember := range storedMembers {
		storedIdentities[storedMember.identity()] = struct{}{}
	}

	return storedIdentities, nil
}
