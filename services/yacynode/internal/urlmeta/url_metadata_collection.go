package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const bucketName vault.Name = "urlmeta"

var urlMetadataKeyLayout = vaultkey.Single(vaultkey.Text)

func registerCollection(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLMetadata], error) {
	collection, err := vault.Register(v, bucketName, storedURLMetadataCodec{})
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}

func urlMetadataKey(hash yacymodel.URLHash) vault.Key {
	return urlMetadataKeyLayout.Key(hash.String()).Bytes()
}
