package remoteindexadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type URLMetadataOutcome string

const (
	URLMetadataStoredNewURL               URLMetadataOutcome = "stored_new_url"
	URLMetadataReplacedOlderVersion       URLMetadataOutcome = "replaced_older_version"
	URLMetadataDroppedAlreadyStored       URLMetadataOutcome = "dropped_already_stored"
	URLMetadataDroppedNotNewerVersion     URLMetadataOutcome = "dropped_not_newer_version"
	URLMetadataDroppedFutureDate          URLMetadataOutcome = "dropped_future_date"
	URLMetadataDroppedTooManyReplacements URLMetadataOutcome = "dropped_too_many_replacements"
)

var _ URLMetadataReceiver = URLMetadataIntake{}

type URLMetadataIntake struct {
	vault         *vault.Vault
	urls          urlmeta.URLDirectory
	evictor       urlmeta.URLEvictor
	urlWriter     urlmeta.URLMetadataWriter
	swapAllowance swapAllowance
	releaser      waitingPostingReleaser
	observer      Observer
	now           func() time.Time
}

func (i URLMetadataIntake) Receive(
	ctx context.Context,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
) (URLMetadataReceipt, error) {
	atCapacity, err := i.vault.AtCapacity(ctx)
	if err != nil {
		return URLMetadataReceipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return URLMetadataReceipt{Busy: true}, nil
	}
	hourlySwapLimit, err := i.swapAllowance.hourlyLimitOf(ctx)
	if err != nil {
		return URLMetadataReceipt{}, err
	}

	var knownURLs int

	err = i.vault.Update(ctx, func(tx *vault.Txn) error {
		counted, err := i.admitEach(ctx, tx, sender, metadata, hourlySwapLimit)
		knownURLs = counted

		return err
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return URLMetadataReceipt{Busy: true}, nil
	}
	if err != nil {
		return URLMetadataReceipt{}, fmt.Errorf("store urls: %w", err)
	}

	return URLMetadataReceipt{KnownURLs: knownURLs}, nil
}

func (i URLMetadataIntake) admitEach(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
	hourlySwapLimit int,
) (int, error) {
	knownURLs := 0
	for _, arrived := range metadata {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("context: %w", err)
		}
		storedVersion, err := i.storedVersionOf(tx, arrived.Hash)
		if err != nil {
			return 0, err
		}
		if storedVersion.Present() {
			knownURLs++
		}
		outcome, err := i.admit(ctx, tx, sender, arrived, storedVersion, hourlySwapLimit)
		if err != nil {
			return 0, err
		}
		tx.RunAfterCommit(func() { i.observer.ObserveURLMetadata(outcome, 1) })
	}

	return knownURLs, nil
}

func (i URLMetadataIntake) storedVersionOf(
	tx *vault.Txn,
	url yacymodel.URLHash,
) (yacymodel.Optional[pageversion.Version], error) {
	storedVersions, err := i.urls.VersionsOf(tx, []yacymodel.URLHash{url})
	if err != nil {
		return yacymodel.None[pageversion.Version](), fmt.Errorf("stored version of url: %w", err)
	}

	return storedVersionOf(storedVersions, url), nil
}

//nolint:revive // argument-limit: each input is one fact the decision reads
func (i URLMetadataIntake) admit(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	storedVersion yacymodel.Optional[pageversion.Version],
	hourlySwapLimit int,
) (URLMetadataOutcome, error) {
	arrivedVersion := pageversion.FromMetadata(arrived, pageversion.SourcePeer(sender))
	version, metadataStored := storedVersion.Get()
	switch {
	case arrivedVersion.IsDatedAfter(i.now()):
		return URLMetadataDroppedFutureDate, nil
	case !metadataStored:
		return i.store(ctx, tx, sender, arrived, arrivedVersion)
	case version.Is(arrivedVersion):
		return URLMetadataDroppedAlreadyStored, nil
	case !version.IsReplacedBy(arrivedVersion):
		return URLMetadataDroppedNotNewerVersion, nil
	}
	swapAllowed, err := i.swapAllowance.allows(tx, sender.Network, hourlySwapLimit)
	if err != nil {
		return "", err
	}
	if !swapAllowed {
		return URLMetadataDroppedTooManyReplacements, nil
	}

	return i.replaceWith(ctx, tx, sender, arrived, arrivedVersion)
}

func (i URLMetadataIntake) store(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	arrivedVersion pageversion.Version,
) (URLMetadataOutcome, error) {
	if err := i.write(ctx, tx, sender, arrived, arrivedVersion); err != nil {
		return "", err
	}

	return URLMetadataStoredNewURL, nil
}

func (i URLMetadataIntake) write(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	arrivedVersion pageversion.Version,
) error {
	if err := i.urlWriter.Write(tx, arrived, pageversion.SourcePeer(sender)); err != nil {
		return err
	}

	return i.releaser.releaseWaitingFor(ctx, tx, arrived.Hash, arrivedVersion)
}

func (i URLMetadataIntake) replaceWith(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	arrivedVersion pageversion.Version,
) (URLMetadataOutcome, error) {
	if err := i.swapAllowance.spend(tx, sender.Network); err != nil {
		return "", err
	}
	if _, err := i.evictor.Purge(ctx, tx, []yacymodel.URLHash{arrived.Hash}); err != nil {
		return "", fmt.Errorf("purge the replaced version: %w", err)
	}
	if err := i.write(ctx, tx, sender, arrived, arrivedVersion); err != nil {
		return "", err
	}

	return URLMetadataReplacedOlderVersion, nil
}
