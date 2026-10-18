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
	if len(encoded) == 0 || encoded[0] != changeFormat {
		return Change{}, fmt.Errorf("%w: unknown format", errBadChange)
	}
	added, rest, err := memberset.MembersFrom(encoded[1:])
	if err != nil {
		return Change{}, err
	}
	removed, rest, err := memberset.MembersFrom(rest)
	if err != nil {
		return Change{}, err
	}
	if len(rest) != 0 {
		return Change{}, fmt.Errorf("%w: %d trailing bytes", errBadChange, len(rest))
	}

	return Change{Added: added, Removed: removed}, nil
}

func (c Change) Encoded() []byte {
	encoded := memberset.AppendMembers([]byte{changeFormat}, c.Added)

	return memberset.AppendMembers(encoded, c.Removed)
}
