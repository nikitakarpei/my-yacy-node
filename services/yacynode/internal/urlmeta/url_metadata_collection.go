package urlmeta

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const bucketName vault.Name = "urlmeta"

func registerCollection(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLMetadata], error) {
	collection, err := vault.Register(v, bucketName, storedURLMetadataCodec{})
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}
