// Package memberset encodes the members of one vault member set in their stored
// order, decodes them one at a time, and keeps member sets in a records bucket for
// engines without merges.
package memberset

import (
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var errBadEncoding = errors.New("bad member set encoding")

func MembersFrom(encoded []byte) (members []vault.StoredMember, rest []byte, err error) {
	decoder, rest, err := DecoderFrom(encoded)
	if err != nil {
		return nil, nil, err
	}

	return decoder.StoredMembers(), rest, nil
}
