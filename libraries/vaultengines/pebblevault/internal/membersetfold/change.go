package membersetfold

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

const changeFormat byte = 1

var errBadChange = errors.New("bad member set change")

type Change struct {
	Added   []vault.StoredMember
	Removed []vault.StoredMember
}

func ChangeFrom(encoded []byte) (Change, error) {
	addedMembers, removedMembers, err := decodersFrom(encoded)
	if err != nil {
		return Change{}, err
	}

	return Change{Added: addedMembers.StoredMembers(), Removed: removedMembers.StoredMembers()}, nil
}

func decodersFrom(encoded []byte) (addedMembers, removedMembers memberset.Decoder, err error) {
	if len(encoded) == 0 || encoded[0] != changeFormat {
		return memberset.Decoder{}, memberset.Decoder{}, fmt.Errorf(
			"%w: unknown format",
			errBadChange,
		)
	}
	addedMembers, rest, err := memberset.DecoderFrom(encoded[1:])
	if err != nil {
		return memberset.Decoder{}, memberset.Decoder{}, err
	}
	removedMembers, rest, err = memberset.DecoderFrom(rest)
	if err != nil {
		return memberset.Decoder{}, memberset.Decoder{}, err
	}
	if len(rest) != 0 {
		return memberset.Decoder{}, memberset.Decoder{}, fmt.Errorf(
			"%w: %d trailing bytes", errBadChange, len(rest),
		)
	}

	return addedMembers, removedMembers, nil
}

func (c Change) Encoded() []byte {
	return encodedChangeOf(memberset.EncoderFrom(c.Added), memberset.EncoderFrom(c.Removed))
}

func encodedChangeOf(added, removed memberset.Encoder) []byte {
	encoded := make([]byte, 0, 1+added.AmountOfBytes()+removed.AmountOfBytes())

	return removed.AppendTo(added.AppendTo(append(encoded, changeFormat)))
}
