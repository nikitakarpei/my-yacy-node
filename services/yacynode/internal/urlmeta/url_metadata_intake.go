package urlmeta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const urlMetadataDiscarded = "url metadata discarded"

var errURLMetadataRefused = errors.New("url metadata refused by storage")

type urlIntake struct {
	vault      *vault.Vault
	collection *vault.Collection[yacymodel.URLHash, yacymodel.URLMetadata]
	observers  observers
}

func (i urlIntake) Receive(
	ctx context.Context,
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
		existing, rejected, storeErr = i.store(ctx, tx, metadata)

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

func (i urlIntake) store(
	ctx context.Context,
	tx *vault.Txn,
	metadata []yacymodel.URLMetadata,
) (existing, rejected []yacymodel.URLHash, err error) {
	for _, stored := range metadata {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("context: %w", err)
		}

		hash := stored.Hash

		_, found, err := i.collection.Get(tx, hash)
		if err != nil {
			return nil, nil, fmt.Errorf("read url metadata: %w", err)
		}
		if found {
			existing = append(existing, hash)
		}
		if err := i.Admit(tx, stored); err != nil {
			if !errors.Is(err, errURLMetadataRefused) {
				return nil, nil, err
			}
			rejected = append(rejected, hash)
			slog.WarnContext(ctx, urlMetadataDiscarded,
				slog.String("reason", "store failed"),
				slog.Any("error", err),
			)

			continue
		}
	}

	return existing, rejected, nil
}

func (i urlIntake) Admit(tx *vault.Txn, metadata yacymodel.URLMetadata) error {
	if _, err := i.collection.Put(tx, metadata.Hash, metadata); err != nil {
		return fmt.Errorf("%w: %w", errURLMetadataRefused, err)
	}

	return i.observers.stored(tx, metadata.Hash, metadata.Freshness())
}

var (
	_ URLMetadataAdmitter = urlIntake{}
	_ URLReceiver         = urlIntake{}
)
