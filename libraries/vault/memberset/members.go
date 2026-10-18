// Package memberset encodes the members of one vault member set, and keeps
// member sets in a records bucket for engines without merges.
package memberset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var errBadEncoding = errors.New("bad member set encoding")

func MembersFrom(encoded []byte) (members []vault.StoredMember, rest []byte, err error) {
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

func idMembersFrom(encoded []byte) ([]vault.StoredMember, []byte, error) {
	amount, rest, err := uvarintFrom(encoded)
	if err != nil {
		return nil, nil, err
	}
	members := make([]vault.StoredMember, 0, min(amount, uint64(len(rest))))
	var previousID uint64
	for range amount {
		var gap uint64
		if gap, rest, err = uvarintFrom(rest); err != nil {
			return nil, nil, err
		}
		previousID += gap
		members = append(members, vault.StoredMember{ID: previousID})
	}

	return members, rest, nil
}

func uvarintFrom(encoded []byte) (uint64, []byte, error) {
	number, length := binary.Uvarint(encoded)
	if length <= 0 {
		return 0, nil, fmt.Errorf("%w: unreadable number", errBadEncoding)
	}

	return number, encoded[length:], nil
}

func rawMembersFrom(encoded []byte) ([]vault.StoredMember, []byte, error) {
	amount, rest, err := uvarintFrom(encoded)
	if err != nil {
		return nil, nil, err
	}
	members := make([]vault.StoredMember, 0, min(amount, uint64(len(rest))))
	for range amount {
		var rawLength uint64
		if rawLength, rest, err = uvarintFrom(rest); err != nil {
			return nil, nil, err
		}
		if rawLength == 0 || rawLength > uint64(len(rest)) {
			return nil, nil, fmt.Errorf("%w: unreadable raw member", errBadEncoding)
		}
		members = append(members, vault.StoredMember{Raw: string(rest[:rawLength])})
		rest = rest[rawLength:]
	}

	return members, rest, nil
}

func AppendMembers(encoded []byte, members []vault.StoredMember) []byte {
	var ids []uint64
	var raws []string
	for _, member := range members {
		if member.Form() == vault.MemberFormRaw {
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
