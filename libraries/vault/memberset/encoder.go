package memberset

import (
	"encoding/binary"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type Encoder struct {
	ids          []byte
	amountOfIDs  uint64
	previousID   uint64
	raws         []byte
	amountOfRaws uint64
}

func EncoderFrom(members []vault.StoredMember) Encoder {
	ids, raws, rawsLength := idsAndRawsOf(members)
	slices.Sort(ids)
	slices.Sort(raws)
	encoder := Encoder{
		ids:  make([]byte, 0, len(ids)*binary.MaxVarintLen32),
		raws: make([]byte, 0, rawsLength+len(raws)*binary.MaxVarintLen16),
	}
	for _, id := range ids {
		encoder.addID(id)
	}
	for _, raw := range raws {
		addRawTo(&encoder, raw)
	}

	return encoder
}

func idsAndRawsOf(members []vault.StoredMember) (ids []uint64, raws []string, rawsLength int) {
	for _, member := range members {
		if member.Form() == vault.MemberFormRaw {
			raws = append(raws, member.Raw)
			rawsLength += len(member.Raw)
		} else {
			ids = append(ids, member.ID)
		}
	}

	return ids, raws, rawsLength
}

func addRawTo[Raw string | []byte](encoder *Encoder, raw Raw) {
	encoder.raws = binary.AppendUvarint(encoder.raws, uint64(len(raw)))
	encoder.raws = append(encoder.raws, raw...)
	encoder.amountOfRaws++
}

func (e *Encoder) addID(id uint64) {
	e.ids = binary.AppendUvarint(e.ids, id-e.previousID)
	e.previousID = id
	e.amountOfIDs++
}

func (e *Encoder) Add(member DecodedMember) {
	if member.Raw == nil {
		e.addID(member.ID)

		return
	}
	addRawTo(e, member.Raw)
}

func EncoderFor(decoders []Decoder) Encoder {
	var idsLength, rawsLength int
	for _, decoder := range decoders {
		idsLength += len(decoder.ids)
		rawsLength += len(decoder.raws)
	}

	return Encoder{ids: make([]byte, 0, idsLength), raws: make([]byte, 0, rawsLength)}
}

func (e *Encoder) AmountOfBytes() int {
	var count [binary.MaxVarintLen64]byte

	return len(binary.AppendUvarint(count[:0], e.amountOfIDs)) + len(e.ids) +
		len(binary.AppendUvarint(count[:0], e.amountOfRaws)) + len(e.raws)
}

func (e *Encoder) AppendTo(encoded []byte) []byte {
	encoded = binary.AppendUvarint(encoded, e.amountOfIDs)
	encoded = append(encoded, e.ids...)
	encoded = binary.AppendUvarint(encoded, e.amountOfRaws)

	return append(encoded, e.raws...)
}
