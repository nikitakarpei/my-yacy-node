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
	added, removed, err := decodersFrom(encoded)
	if err != nil {
		return Change{}, err
	}

	return Change{Added: added.StoredMembers(), Removed: removed.StoredMembers()}, nil
}

func decodersFrom(encoded []byte) (added, removed memberset.Decoder, err error) {
	if len(encoded) == 0 || encoded[0] != changeFormat {
		return memberset.Decoder{}, memberset.Decoder{}, fmt.Errorf(
			"%w: unknown format",
			errBadChange,
		)
	}
	added, rest, err := memberset.DecoderFrom(encoded[1:])
	if err != nil {
		return memberset.Decoder{}, memberset.Decoder{}, err
	}
	removed, rest, err = memberset.DecoderFrom(rest)
	if err != nil {
		return memberset.Decoder{}, memberset.Decoder{}, err
	}
	if len(rest) != 0 {
		return memberset.Decoder{}, memberset.Decoder{}, fmt.Errorf(
			"%w: %d trailing bytes", errBadChange, len(rest),
		)
	}

	return added, removed, nil
}

func (c Change) Encoded() []byte {
	return encodedChangeOf(memberset.EncoderFrom(c.Added), memberset.EncoderFrom(c.Removed))
}

func encodedChangeOf(added, removed memberset.Encoder) []byte {
	encoded := make([]byte, 0, 1+added.AmountOfBytes()+removed.AmountOfBytes())

	return removed.AppendTo(added.AppendTo(append(encoded, changeFormat)))
}
