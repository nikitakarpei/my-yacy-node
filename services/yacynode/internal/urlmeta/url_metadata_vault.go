package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const bucketName vault.Name = "urlmeta"

func registerCollection(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLHash, yacymodel.URLMetadata], error) {
	collection, err := vault.RegisterCollection(
		v,
		bucketName,
		urlMetadataKeyCodec{},
		urlMetadataValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}

var urlMetadataKeyLayout = vaultkey.Single(hashcodec.URLHash)

type urlMetadataKeyCodec struct{}

func (urlMetadataKeyCodec) Encode(hash yacymodel.URLHash) vaultkey.Key {
	return urlMetadataKeyLayout.Key(hash)
}

func (urlMetadataKeyCodec) Decode(key vaultkey.Key) (yacymodel.URLHash, error) {
	hash, err := urlMetadataKeyLayout.Parts(key)
	if err != nil {
		return yacymodel.URLHash{}, fmt.Errorf("url metadata key: %w", err)
	}

	return hash, nil
}
