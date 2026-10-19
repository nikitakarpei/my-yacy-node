package memberset

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type Decoder struct {
	ids          []byte
	amountOfIDs  uint64
	previousID   uint64
	raws         []byte
	amountOfRaws uint64
}

func DecoderFrom(encoded []byte) (Decoder, []byte, error) {
	var decoder Decoder
	var err error
	if decoder.amountOfIDs, decoder.ids, encoded, err = idsFrom(encoded); err != nil {
		return Decoder{}, nil, err
	}
	if decoder.amountOfRaws, decoder.raws, encoded, err = rawsFrom(encoded); err != nil {
		return Decoder{}, nil, err
	}

	return decoder, encoded, nil
}

func idsFrom(encoded []byte) (amountOfIDs uint64, ids, rest []byte, err error) {
	amountOfIDs, rest, err = uvarintFrom(encoded)
	if err != nil {
		return 0, nil, nil, err
	}
	idsLength := 0
	for range amountOfIDs {
		_, gapLength := binary.Uvarint(rest[idsLength:])
		if gapLength <= 0 {
			return 0, nil, nil, fmt.Errorf("%w: unreadable number", errBadEncoding)
		}
		idsLength += gapLength
	}

	return amountOfIDs, rest[:idsLength], rest[idsLength:], nil
}

func uvarintFrom(encoded []byte) (uint64, []byte, error) {
	number, length := binary.Uvarint(encoded)
	if length <= 0 {
		return 0, nil, fmt.Errorf("%w: unreadable number", errBadEncoding)
	}

	return number, encoded[length:], nil
}

func rawsFrom(encoded []byte) (amountOfRaws uint64, raws, rest []byte, err error) {
	amountOfRaws, rest, err = uvarintFrom(encoded)
	if err != nil {
		return 0, nil, nil, err
	}
	rawsLength := 0
	for range amountOfRaws {
		rawLength, lengthLength := binary.Uvarint(rest[rawsLength:])
		if lengthLength <= 0 {
			return 0, nil, nil, fmt.Errorf("%w: unreadable number", errBadEncoding)
		}
		//nolint:gosec // G115: a length of bytes left is never negative.
		if rawLength == 0 || rawLength > uint64(len(rest)-rawsLength-lengthLength) {
			return 0, nil, nil, fmt.Errorf("%w: unreadable raw member", errBadEncoding)
		}
		//nolint:gosec // G115: rawLength is at most the length of rest.
		rawsLength += lengthLength + int(rawLength)
	}

	return amountOfRaws, rest[:rawsLength], rest[rawsLength:], nil
}

func (d Decoder) StoredMembers() []vault.StoredMember {
	members := make([]vault.StoredMember, 0, d.AmountOfMembers())
	for member, found := d.Next(); found; member, found = d.Next() {
		members = append(members, member.Stored())
	}

	return members
}

func (d Decoder) AmountOfMembers() int {
	//nolint:gosec // G115: every member takes at least one encoded byte.
	return int(d.amountOfIDs + d.amountOfRaws)
}

func (d *Decoder) Next() (DecodedMember, bool) {
	if d.amountOfIDs > 0 {
		gap, gapLength := binary.Uvarint(d.ids)
		d.ids = d.ids[gapLength:]
		d.amountOfIDs--
		d.previousID += gap

		return DecodedMember{ID: d.previousID}, true
	}
	if d.amountOfRaws > 0 {
		rawLength, lengthLength := binary.Uvarint(d.raws)
		//nolint:gosec // G115: DecoderFrom checked every raw length against the encoding.
		rawEnd := lengthLength + int(rawLength)
		raw := d.raws[lengthLength:rawEnd:rawEnd]
		d.raws = d.raws[rawEnd:]
		d.amountOfRaws--

		return DecodedMember{Raw: raw}, true
	}

	return DecodedMember{}, false
}

type DecodedMember struct {
	ID  uint64
	Raw []byte
}

func (m DecodedMember) Stored() vault.StoredMember {
	if m.Raw == nil {
		return vault.StoredMember{ID: m.ID}
	}

	return vault.StoredMember{Raw: string(m.Raw)}
}

func (m DecodedMember) CompareInStoredOrder(other DecodedMember) int {
	switch {
	case m.Raw == nil && other.Raw == nil:
		return cmp.Compare(m.ID, other.ID)
	case m.Raw == nil:
		return -1
	case other.Raw == nil:
		return 1
	default:
		return bytes.Compare(m.Raw, other.Raw)
	}
}
