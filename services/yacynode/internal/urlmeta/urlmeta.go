// Package urlmeta owns the transferURL endpoint, URL intake, and URL metadata
// storage and lookup. Its published ports speak the yacymodel vocabulary and
// never leak the schema. A port that reads or writes stored metadata takes the
// transaction its caller opened, so the caller decides and writes in one
// transaction.
package urlmeta

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type URLDirectory interface {
	MetadataPerHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]yacymodel.URLMetadata, error)
	MissingURLs(tx *vault.Txn, hashes []yacymodel.URLHash) ([]yacymodel.URLHash, error)
	Count(tx *vault.Txn) (int, error)
}

type URLReceiver interface {
	Receive(ctx context.Context, metadata []yacymodel.URLMetadata) (Receipt, error)
}

type URLEvictor interface {
	Purge(ctx context.Context, tx *vault.Txn, urls []yacymodel.URLHash) (PurgeResult, error)
}

type URLMetadataObserver interface {
	URLStored(
		tx *vault.Txn,
		hash yacymodel.URLHash,
		freshness yacymodel.Optional[yacymodel.CalendarDay],
	) error
	URLPurged(tx *vault.Txn, hash yacymodel.URLHash) error
}

type Receipt struct {
	Busy     bool
	Double   int
	ErrorURL []yacymodel.URLHash
}

type PurgeResult struct {
	URLsDeleted int
}

func Open(
	vault *vault.Vault,
	watchers ...URLMetadataObserver,
) (URLDirectory, URLEvictor, URLReceiver, error) {
	collection, err := registerCollection(vault)
	if err != nil {
		return nil, nil, nil, err
	}

	watched := observers(watchers)
	directory := urlDirectory{collection: collection, observers: watched}

	return directory,
		directory,
		urlIntake{vault: vault, collection: collection, observers: watched},
		nil
}

func MountTransferURL(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	receiver URLReceiver,
) {
	httpguard.Mount(
		router,
		yacyproto.PathTransferURL,
		yacyproto.TransferURLEndpointMethods,
		yacyproto.ParseTransferURLRequest,
		transferURLEndpoint{identity: identity, intake: receiver}.Serve,
	)
}
