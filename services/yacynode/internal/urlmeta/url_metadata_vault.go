package urlmeta

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta/urlmetarecord"
)

const urlMetadataBucket vault.Name = "urlmeta"

func registerCollection(
	ctx context.Context,
	v *vault.Vault,
	projections []vault.Projection[yacymodel.URLMetadata],
) (*vault.Collection[yacymodel.URLHash, urlmetarecord.Record], error) {
	collection, err := v.RegisterProjectedCollection(
		ctx,
		urlMetadataBucket,
		urlMetadataKeyLayout,
		urlmetarecord.Codec{},
		vault.ProjectedAs(metadataWithHash, projections...),
	)
	if err != nil {
		return nil, fmt.Errorf("register url metadata collection: %w", err)
	}

	return collection, nil
}

func metadataWithHash(
	hash yacymodel.URLHash,
	stored urlmetarecord.Record,
) yacymodel.URLMetadata {
	metadata := stored.Metadata
	metadata.Hash = hash

	return metadata
}

var urlMetadataKeyLayout = vault.SingleKey(hashkeypart.URLHash).KeyLayout()
