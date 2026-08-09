package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var urlMetadataKeyLayout = vaultkey.Single(vaultkey.Text)

type urlMetadataKeyCodec struct{}

func (urlMetadataKeyCodec) Encode(hash yacymodel.URLHash) vaultkey.Key {
	return urlMetadataKeyLayout.Key(hash.String())
}

func (urlMetadataKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	hash, err := urlMetadataKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("url metadata key: %w", err)
	}

	return yacymodel.ParseURLHash(hash)
}
