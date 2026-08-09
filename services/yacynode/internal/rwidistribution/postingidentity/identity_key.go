package postingidentity

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var identityKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type KeyCodec struct{}

func (KeyCodec) Encode(identity Identity) vaultkey.Key {
	return identityKeyLayout.Key(identity.Word.String(), identity.URL.String())
}

func (KeyCodec) Decode(key vaultkey.Key) (Identity, error) {
	word, url, err := identityKeyLayout.Parts(key)
	if err != nil {
		return Identity{}, fmt.Errorf("posting identity key: %w", err)
	}

	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return Identity{}, fmt.Errorf("posting identity word hash: %w", err)
	}
	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return Identity{}, fmt.Errorf("posting identity url hash: %w", err)
	}

	return IdentityOf(parsedWord, parsedURL), nil
}
