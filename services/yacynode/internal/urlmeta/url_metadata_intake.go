package urlmeta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const urlMetadataDiscarded = "url metadata discarded"

var errURLMetadataRefused = errors.New("url metadata refused by storage")

type urlIntake struct {
	vault         *vault.Vault
	collection    *vault.Collection[yacymodel.URLHash, storedURLMetadata]
	directory     urlDirectory
	swapAllowance swapAllowance
	observers     observers
	observer      Observer
	now           func() time.Time
}

func (i urlIntake) Receive(
	ctx context.Context,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
) (Receipt, error) {
	atCapacity, err := i.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return Receipt{Busy: true}, nil
	}

	var existing, rejected []yacymodel.URLHash

	err = i.vault.Update(ctx, func(tx *vault.Txn) error {
		var storeErr error
		existing, rejected, storeErr = i.placeEach(ctx, tx, sender, metadata)

		return storeErr
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return Receipt{Busy: true}, nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("store urls: %w", err)
	}

	return Receipt{Double: len(existing), ErrorURL: rejected}, nil
}

func (i urlIntake) placeEach(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	metadata []yacymodel.URLMetadata,
) (existing, rejected []yacymodel.URLHash, err error) {
	for _, arrived := range metadata {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("context: %w", err)
		}

		outcome, err := i.outcomeOf(tx, sender, arrived)
		if err != nil {
			return nil, nil, err
		}
		if outcome != OutcomeStoredNewURL && outcome != OutcomeDroppedFutureDate {
			existing = append(existing, arrived.Hash)
		}
		if err := i.carryOut(ctx, tx, sender, arrived, outcome); err != nil {
			if !errors.Is(err, errURLMetadataRefused) || outcome == OutcomeReplacedOlderVersion {
				return nil, nil, err
			}
			rejected = append(rejected, arrived.Hash)
			slog.WarnContext(ctx, urlMetadataDiscarded,
				slog.String("reason", "store failed"),
				slog.Any("error", err),
			)

			continue
		}
		tx.RunAfterCommit(func() { i.observer.ObserveOutcome(outcome, 1) })
	}

	return existing, rejected, nil
}

func (i urlIntake) outcomeOf(
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) (Outcome, error) {
	arrivedVersion := pagefingerprint.FromMetadata(arrived)
	if arrivedVersion.IsDatedAfter(i.now()) {
		return OutcomeDroppedFutureDate, nil
	}
	stored, found, err := i.collection.Get(tx, arrived.Hash)
	if err != nil {
		return "", fmt.Errorf("read url metadata: %w", err)
	}
	if !found {
		return OutcomeStoredNewURL, nil
	}
	storedVersion := pagefingerprint.FromMetadata(stored.metadata)
	switch {
	case arrivedVersion == storedVersion:
		return OutcomeDroppedAlreadyStored, nil
	case arrivedVersion.Modified <= storedVersion.Modified:
		return OutcomeDroppedOtherVersion, nil
	}
	allowed, err := i.swapAllowance.allows(tx, sender.Network)
	if err != nil {
		return "", err
	}
	if !allowed {
		return OutcomeDroppedOverSwapLimit, nil
	}

	return OutcomeReplacedOlderVersion, nil
}

func (i urlIntake) carryOut(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
	outcome Outcome,
) error {
	switch outcome {
	case OutcomeStoredNewURL:
		return i.put(tx, storedURLMetadata{metadata: arrived, source: SourcePeer(sender)})
	case OutcomeReplacedOlderVersion:
		return i.replaceWith(ctx, tx, sender, arrived)
	default:
		return nil
	}
}

func (i urlIntake) put(tx *vault.Txn, stored storedURLMetadata) error {
	if _, err := i.collection.Put(tx, stored.metadata.Hash, stored); err != nil {
		return fmt.Errorf("%w: %w", errURLMetadataRefused, err)
	}

	return i.observers.stored(tx, stored)
}

func (i urlIntake) replaceWith(
	ctx context.Context,
	tx *vault.Txn,
	sender rwiorigin.Origin,
	arrived yacymodel.URLMetadata,
) error {
	if err := i.swapAllowance.spend(tx, sender.Network); err != nil {
		return err
	}
	if _, err := i.directory.Purge(ctx, tx, []yacymodel.URLHash{arrived.Hash}); err != nil {
		return err
	}

	return i.put(tx, storedURLMetadata{metadata: arrived, source: SourcePeer(sender)})
}

func (i urlIntake) Admit(tx *vault.Txn, metadata yacymodel.URLMetadata) error {
	return i.put(tx, storedURLMetadata{metadata: metadata, source: SourceThisNode()})
}

var (
	_ URLMetadataAdmitter = urlIntake{}
	_ URLReceiver         = urlIntake{}
)
