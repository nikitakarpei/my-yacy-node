package postingidentity

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
)

var identityKeyLayout = vault.PairKey(hashcodec.Hash, hashcodec.URLHash)

type KeyCodec struct{}

func (KeyCodec) Encode(identity Identity) vault.Key {
	return identityKeyLayout.Key(identity.Word, identity.URL)
}

func (KeyCodec) Decode(storedKey []byte) (Identity, error) {
	word, url, err := identityKeyLayout.Parts(storedKey)
	if err != nil {
		return Identity{}, fmt.Errorf("posting identity key: %w", err)
	}

	return IdentityOf(word, url), nil
}
