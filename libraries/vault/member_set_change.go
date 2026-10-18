package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

const memberSetFormat byte = 1

var errBadMemberSet = errors.New("bad member set")

type MemberSetChange struct {
	Added   []StoredMember
	Removed []StoredMember
}

func MemberSetChangeFrom(encoded []byte) (MemberSetChange, error) {
	if len(encoded) == 0 || encoded[0] != memberSetFormat {
		return MemberSetChange{}, fmt.Errorf("%w: unknown format", errBadMemberSet)
	}
	added, rest, err := membersFrom(encoded[1:])
	if err != nil {
		return MemberSetChange{}, err
	}
	removed, rest, err := membersFrom(rest)
	if err != nil {
		return MemberSetChange{}, err
	}
	if len(rest) != 0 {
		return MemberSetChange{}, fmt.Errorf("%w: %d trailing bytes", errBadMemberSet, len(rest))
	}

	return MemberSetChange{Added: added, Removed: removed}, nil
}

func StoredMembersFrom(encoded []byte) ([]StoredMember, error) {
	change, err := MemberSetChangeFrom(encoded)
	if err != nil {
		return nil, fmt.Errorf("read member set: %w", err)
	}

	return change.Added, nil
}

func membersFrom(encoded []byte) ([]StoredMember, []byte, error) {
	idMembers, rest, err := idMembersFrom(encoded)
	if err != nil {
		return nil, nil, err
	}
	rawMembers, rest, err := rawMembersFrom(rest)
	if err != nil {
		return nil, nil, err
	}

	return append(idMembers, rawMembers...), rest, nil
}

func idMembersFrom(encoded []byte) ([]StoredMember, []byte, error) {
	amount, rest, err := uvarintFrom(encoded)
	if err != nil {
		return nil, nil, err
	}
	members := make([]StoredMember, 0, min(amount, uint64(len(rest))))
	var previousID uint64
	for range amount {
		var gap uint64
		if gap, rest, err = uvarintFrom(rest); err != nil {
			return nil, nil, err
		}
		previousID += gap
		members = append(members, StoredMember{ID: previousID})
	}

	return members, rest, nil
}

func uvarintFrom(encoded []byte) (uint64, []byte, error) {
	number, length := binary.Uvarint(encoded)
	if length <= 0 {
		return 0, nil, fmt.Errorf("%w: unreadable number", errBadMemberSet)
	}

	return number, encoded[length:], nil
}

func rawMembersFrom(encoded []byte) ([]StoredMember, []byte, error) {
	amount, rest, err := uvarintFrom(encoded)
	if err != nil {
		return nil, nil, err
	}
	members := make([]StoredMember, 0, min(amount, uint64(len(rest))))
	for range amount {
		var rawLength uint64
		if rawLength, rest, err = uvarintFrom(rest); err != nil {
			return nil, nil, err
		}
		if rawLength == 0 || rawLength > uint64(len(rest)) {
			return nil, nil, fmt.Errorf("%w: unreadable raw member", errBadMemberSet)
		}
		members = append(members, StoredMember{Raw: string(rest[:rawLength])})
		rest = rest[rawLength:]
	}

	return members, rest, nil
}

func (c MemberSetChange) Encoded() []byte {
	encoded := appendMembers([]byte{memberSetFormat}, c.Added)

	return appendMembers(encoded, c.Removed)
}

func appendMembers(encoded []byte, members []StoredMember) []byte {
	var ids []uint64
	var raws []string
	for _, member := range members {
		if member.form() == MemberFormRaw {
			raws = append(raws, member.Raw)
		} else {
			ids = append(ids, member.ID)
		}
	}

	return appendRaws(appendIDs(encoded, ids), raws)
}

func appendIDs(encoded []byte, ids []uint64) []byte {
	slices.Sort(ids)
	encoded = binary.AppendUvarint(encoded, uint64(len(ids)))
	var previousID uint64
	for _, id := range ids {
		encoded = binary.AppendUvarint(encoded, id-previousID)
		previousID = id
	}

	return encoded
}

func appendRaws(encoded []byte, raws []string) []byte {
	slices.Sort(raws)
	encoded = binary.AppendUvarint(encoded, uint64(len(raws)))
	for _, raw := range raws {
		encoded = binary.AppendUvarint(encoded, uint64(len(raw)))
		encoded = append(encoded, raw...)
	}

	return encoded
}
