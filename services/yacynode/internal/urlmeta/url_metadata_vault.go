package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const bucketName vault.Name = "urlmeta"

func registerCollection(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLHash, yacymodel.URLMetadata], error) {
	collection, err := v.RegisterCollection(
		bucketName,
		urlMetadataKeyLayout,
		urlMetadataValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}

var urlMetadataKeyLayout = vault.SingleKey(hashkeypart.URLHash).KeyLayout()
