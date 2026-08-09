package urlmeta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const urlMetadataDiscarded = "url metadata discarded"

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

		hash, err := stored.Hash()
		if err != nil {
			slog.WarnContext(ctx, urlMetadataDiscarded,
				slog.String("reason", "invalid url hash"),
				slog.Any("error", err),
			)

			continue
		}

		_, found, err := i.collection.Get(tx, hash)
		if err != nil {
			return nil, nil, fmt.Errorf("read url metadata: %w", err)
		}
		if found {
			existing = append(existing, hash)
		}
		if err := i.collection.Put(tx, hash, stored); err != nil {
			rejected = append(rejected, hash)
			slog.WarnContext(ctx, urlMetadataDiscarded,
				slog.String("reason", "store failed"),
				slog.Any("error", err),
			)

			continue
		}
		i.observers.stored(ctx, tx, hash, stored.Freshness())
	}

	return existing, rejected, nil
}
