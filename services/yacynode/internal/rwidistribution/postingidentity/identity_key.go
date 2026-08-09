package postingidentity

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var identityKeyLayout = vaultkey.Pair(hashcodec.Hash, hashcodec.URLHash)

type KeyCodec struct{}

func (KeyCodec) Encode(identity Identity) vaultkey.Key {
	return identityKeyLayout.Key(identity.Word, identity.URL)
}

func (KeyCodec) Decode(key vaultkey.Key) (Identity, error) {
	word, url, err := identityKeyLayout.Parts(key)
	if err != nil {
		return Identity{}, fmt.Errorf("posting identity key: %w", err)
	}

	return IdentityOf(word, url), nil
}
