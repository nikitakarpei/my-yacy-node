package remoteindexadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
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
	postings      PostingAdmission
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
) (yacymodel.Optional[urlmeta.CurrentVersion], error) {
	currentVersions, err := i.urls.CurrentVersionsOf(tx, []yacymodel.URLHash{url})
	if err != nil {
		return yacymodel.None[urlmeta.CurrentVersion](), fmt.Errorf(
			"current version of url: %w",
			err,
		)
	}

	return storedVersionOf(currentVersions, url), nil
}

//nolint:revive // argument-limit: each input is one fact the decision reads
func (i URLMetadataIntake) admit(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	storedVersion yacymodel.Optional[urlmeta.CurrentVersion],
	hourlySwapLimit int,
) (URLMetadataOutcome, error) {
	arrivedFingerprint := pagefingerprint.FromMetadata(arrived)
	currentVersion, metadataStored := storedVersion.Get()
	switch {
	case arrivedFingerprint.IsDatedAfter(i.now()):
		return URLMetadataDroppedFutureDate, nil
	case !metadataStored:
		return i.store(tx, sender, arrived)
	case currentVersion.Describes(arrivedFingerprint):
		return URLMetadataDroppedAlreadyStored, nil
	case !currentVersion.IsOlderThan(arrivedFingerprint):
		return URLMetadataDroppedNotNewerVersion, nil
	}
	swapAllowed, err := i.swapAllowance.allows(tx, sender.Network, hourlySwapLimit)
	if err != nil {
		return "", err
	}
	if !swapAllowed {
		return URLMetadataDroppedTooManyReplacements, nil
	}

	return i.replaceWith(ctx, tx, sender, arrived)
}

func (i URLMetadataIntake) store(
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) (URLMetadataOutcome, error) {
	if err := i.write(tx, sender, arrived); err != nil {
		return "", err
	}

	return URLMetadataStoredNewURL, nil
}

func (i URLMetadataIntake) write(
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) error {
	currentVersion, err := i.urlWriter.Write(tx, arrived, urlmeta.SourcePeer(sender))
	if err != nil {
		return err
	}

	return i.postings.releaseWaitingFor(tx, arrived.Hash, currentVersion)
}

func (i URLMetadataIntake) replaceWith(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) (URLMetadataOutcome, error) {
	if err := i.swapAllowance.spend(tx, sender.Network); err != nil {
		return "", err
	}
	if _, err := i.evictor.Purge(ctx, tx, []yacymodel.URLHash{arrived.Hash}); err != nil {
		return "", fmt.Errorf("purge the replaced version: %w", err)
	}
	if err := i.write(tx, sender, arrived); err != nil {
		return "", err
	}

	return URLMetadataReplacedOlderVersion, nil
}
