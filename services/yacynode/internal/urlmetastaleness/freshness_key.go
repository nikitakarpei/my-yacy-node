package urlmetastaleness

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var freshnessKeyLayout = vaultkey.Single(vaultkey.Text)

type freshnessKeyCodec struct{}

func (freshnessKeyCodec) Encode(hash yacymodel.URLHash) vaultkey.Key {
	return freshnessKeyLayout.Key(hash.String())
}

func (freshnessKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	hash, err := freshnessKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("staleness freshness key: %w", err)
	}

	return yacymodel.ParseURLHash(hash)
}
