package urlmeta

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const bucketName vault.Name = "urlmeta"

type storedURLMetadata struct {
	metadata yacymodel.URLMetadata
	source   Source
}

func registerCollection(
	ctx context.Context,
	v *vault.Vault,
	projections []vault.Projection[yacymodel.URLMetadata],
) (*vault.Collection[yacymodel.URLHash, storedURLMetadata], error) {
	collection, err := v.RegisterProjectedCollection(
		ctx,
		bucketName,
		urlMetadataKeyLayout,
		urlMetadataValueCodec{},
		vault.ProjectedAs(metadataWithHash, projections...),
	)
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}

func metadataWithHash(
	hash yacymodel.URLHash,
	stored storedURLMetadata,
) yacymodel.URLMetadata {
	metadata := stored.metadata
	metadata.Hash = hash

	return metadata
}

var urlMetadataKeyLayout = vault.SingleKey(hashkeypart.URLHash).KeyLayout()
