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
	URLMetadataStoredNewURL         URLMetadataOutcome = "stored_new_url"
	URLMetadataReplacedOlderVersion URLMetadataOutcome = "replaced_older_version"
	URLMetadataDroppedAlreadyStored URLMetadataOutcome = "dropped_already_stored"
	URLMetadataDroppedOtherVersion  URLMetadataOutcome = "dropped_other_version"
	URLMetadataDroppedFutureDate    URLMetadataOutcome = "dropped_future_date"
	URLMetadataDroppedOverSwapLimit URLMetadataOutcome = "dropped_over_swap_limit"
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

	var double int

	err = i.vault.Update(ctx, func(tx *vault.Txn) error {
		known, err := i.placeEach(ctx, tx, sender, metadata)
		double = known

		return err
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return URLMetadataReceipt{Busy: true}, nil
	}
	if err != nil {
		return URLMetadataReceipt{}, fmt.Errorf("store urls: %w", err)
	}

	return URLMetadataReceipt{Double: double}, nil
}

func (i URLMetadataIntake) placeEach(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
) (int, error) {
	known := 0
	for _, arrived := range metadata {
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("context: %w", err)
		}

		outcome, err := i.outcomeFor(tx, sender, arrived)
		if err != nil {
			return 0, err
		}
		if outcome != URLMetadataStoredNewURL && outcome != URLMetadataDroppedFutureDate {
			known++
		}
		if err := i.carryOut(ctx, tx, sender, arrived, outcome); err != nil {
			return 0, err
		}
		tx.RunAfterCommit(func() { i.observer.ObserveURLMetadata(outcome, 1) })
	}

	return known, nil
}

func (i URLMetadataIntake) outcomeFor(
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) (URLMetadataOutcome, error) {
	arrivedFingerprint := pagefingerprint.FromMetadata(arrived)
	if arrivedFingerprint.IsDatedAfter(i.now()) {
		return URLMetadataDroppedFutureDate, nil
	}
	currentVersions, err := i.urls.CurrentVersionsOf(tx, []yacymodel.URLHash{arrived.Hash})
	if err != nil {
		return "", fmt.Errorf("current version of url: %w", err)
	}
	currentVersion, found := currentVersions[arrived.Hash]
	switch {
	case !found:
		return URLMetadataStoredNewURL, nil
	case arrivedFingerprint == currentVersion.Fingerprint:
		return URLMetadataDroppedAlreadyStored, nil
	case arrivedFingerprint.Modified <= currentVersion.Fingerprint.Modified:
		return URLMetadataDroppedOtherVersion, nil
	}
	allowed, err := i.swapAllowance.allows(tx, sender.Network)
	if err != nil {
		return "", err
	}
	if !allowed {
		return URLMetadataDroppedOverSwapLimit, nil
	}

	return URLMetadataReplacedOlderVersion, nil
}

func (i URLMetadataIntake) carryOut(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	outcome URLMetadataOutcome,
) error {
	switch outcome {
	case URLMetadataStoredNewURL:
		return i.store(tx, sender, arrived)
	case URLMetadataReplacedOlderVersion:
		return i.replaceWith(ctx, tx, sender, arrived)
	default:
		return nil
	}
}

func (i URLMetadataIntake) store(
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) error {
	currentVersion, err := i.urlWriter.Write(tx, arrived, urlmeta.SourcePeer(sender))
	if err != nil {
		return err
	}

	return i.postings.placeWaitingFor(tx, arrived.Hash, currentVersion)
}

func (i URLMetadataIntake) replaceWith(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) error {
	if err := i.swapAllowance.spend(tx, sender.Network); err != nil {
		return err
	}
	if _, err := i.evictor.Purge(ctx, tx, []yacymodel.URLHash{arrived.Hash}); err != nil {
		return fmt.Errorf("purge the replaced version: %w", err)
	}

	return i.store(tx, sender, arrived)
}
