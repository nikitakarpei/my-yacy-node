// Package postingidentity names one posting by its word and URL hash, and
// gives that identity a stable on-disk key.
package postingidentity

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Identity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

func IdentityOf(posting yacymodel.RWIPosting) Identity {
	return Identity{Word: posting.WordHash, URL: posting.URLHash}
}
