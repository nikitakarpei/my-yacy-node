// Package urlmeta owns URL metadata storage and lookup, and the urls.xml
// endpoint. Its published ports speak the yacymodel vocabulary and never leak
// the schema. A port that reads or writes stored metadata takes the transaction
// its caller opened, so the caller decides and writes in one transaction.
package urlmeta

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type URLDirectory interface {
	MetadataPerHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]yacymodel.URLMetadata, error)
	VersionsOf(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]pageversion.Version, error)
	Count(tx *vault.Txn) (int, error)
}

type URLMetadataWriter interface {
	Write(tx *vault.Txn, metadata yacymodel.URLMetadata, source pageversion.Source) error
}

type URLEvictor interface {
	Purge(ctx context.Context, tx *vault.Txn, urls []yacymodel.URLHash) (PurgeResult, error)
}

type URLPurgedObserver interface {
	URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error
}

type PurgeResult struct {
	URLsDeleted int
}

func Open(
	ctx context.Context,
	vault *vault.Vault,
	projections []vault.Projection[yacymodel.URLMetadata],
	watchers ...URLPurgedObserver,
) (URLDirectory, URLEvictor, URLMetadataWriter, error) {
	collection, err := registerCollection(ctx, vault, projections)
	if err != nil {
		return nil, nil, nil, err
	}

	directory := urlDirectory{collection: collection, observers: observers(watchers)}

	return directory, directory, directory, nil
}

func MountURLMetadataLookup(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	vault *vault.Vault,
	directory URLDirectory,
	servedURLMetadataPerRequest int,
) {
	httpguard.MountFeed(
		router,
		yacyproto.PathURLMetadata,
		yacyproto.URLMetadataEndpointMethods,
		yacyproto.ParseURLMetadataRequest,
		urlMetadataLookupEndpoint{
			identity:                    identity,
			vault:                       vault,
			directory:                   directory,
			servedURLMetadataPerRequest: servedURLMetadataPerRequest,
		}.Serve,
	)
}
