// Package postingidentity names one posting by its word and URL hash, and
// gives that identity a stable on-disk key.
package postingidentity

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var identityKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type Identity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

func IdentityOf(word yacymodel.Hash, url yacymodel.URLHash) Identity {
	return Identity{Word: word, URL: url}
}

func (i Identity) Key() vault.Key {
	return identityKeyLayout.Key(i.Word.String(), i.URL.String()).Bytes()
}
