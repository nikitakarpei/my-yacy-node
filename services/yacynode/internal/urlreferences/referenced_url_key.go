package urlreferences

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var referencedURLKeyLayout = vaultkey.Single(vaultkey.Text)

type referencedURLKeyCodec struct{}

func (referencedURLKeyCodec) Encode(url yacymodel.URLHash) vaultkey.Key {
	return referencedURLKeyLayout.Key(url.String())
}

func (referencedURLKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	url, err := referencedURLKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("referenced url key: %w", err)
	}

	return yacymodel.ParseURLHash(url)
}
