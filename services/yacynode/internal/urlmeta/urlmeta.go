// Package urlmeta owns the transferURL and urls.xml endpoints, URL intake, and
// URL metadata storage and lookup. Its published ports speak the yacymodel vocabulary and
// never leak the schema. A port that reads or writes stored metadata takes the
// transaction its caller opened, so the caller decides and writes in one
// transaction. Each record knows its source: this node or the network of the
// peer that sent it. A later version from a peer replaces the stored metadata
// and every posting of its URL in one transaction, within an hourly limit per
// network.
package urlmeta

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type URLDirectory interface {
	MetadataPerHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]yacymodel.URLMetadata, error)
	CurrentVersionsOf(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]CurrentVersion, error)
	Count(tx *vault.Txn) (int, error)
}

type URLReceiver interface {
	Receive(
		ctx context.Context,
		sender rwiorigin.Origin,
		metadata []yacymodel.URLMetadata,
	) (Receipt, error)
}

type URLMetadataAdmitter interface {
	Admit(tx *vault.Txn, metadata yacymodel.URLMetadata) error
}

type URLEvictor interface {
	Purge(ctx context.Context, tx *vault.Txn, urls []yacymodel.URLHash) (PurgeResult, error)
}

type URLMetadataObserver interface {
	URLStored(
		tx *vault.Txn,
		hash yacymodel.URLHash,
		version CurrentVersion,
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

type Outcome string

const (
	OutcomeStoredNewURL         Outcome = "stored_new_url"
	OutcomeReplacedOlderVersion Outcome = "replaced_older_version"
	OutcomeDroppedAlreadyStored Outcome = "dropped_already_stored"
	OutcomeDroppedOtherVersion  Outcome = "dropped_other_version"
	OutcomeDroppedFutureDate    Outcome = "dropped_future_date"
	OutcomeDroppedOverSwapLimit Outcome = "dropped_over_swap_limit"
)

type Observer interface {
	ObserveOutcome(outcome Outcome, records int)
}

type SwapLimit struct {
	FloorPerHour int
	SharePerHour float64
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	ctx context.Context,
	vault *vault.Vault,
	now func() time.Time,
	swapLimit SwapLimit,
	observer Observer,
	projections []vault.Projection[yacymodel.URLMetadata],
	watchers ...URLMetadataObserver,
) (URLDirectory, URLEvictor, URLMetadataAdmitter, URLReceiver, error) {
	collection, err := registerCollection(ctx, vault, projections)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	windows, err := registerSwapWindows(vault)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	watched := observers(watchers)
	directory := urlDirectory{collection: collection, observers: watched}
	intake := urlIntake{
		vault:      vault,
		collection: collection,
		directory:  directory,
		swapAllowance: swapAllowance{
			windows:    windows,
			collection: collection,
			now:        now,
			limit:      swapLimit,
		},
		observers: watched,
		observer:  observer,
		now:       now,
	}

	return directory, directory, intake, intake, nil
}

func MountTransferURL(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	receiver URLReceiver,
) {
	httpguard.MountFields(
		router,
		yacyproto.PathTransferURL,
		yacyproto.TransferURLEndpointMethods,
		yacyproto.ParseTransferURLRequest,
		transferURLEndpoint{identity: identity, receiver: receiver}.Serve,
	)
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
