package rwipostings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type postingIntake struct {
	vault     *vault.Vault
	postings  *vault.Collection[yacymodel.RWIPosting]
	observers postingObservers
	urls      urlmeta.URLDirectory
	batchCap  int
	pause     time.Duration
}

func (i postingIntake) Receive(
	ctx context.Context,
	entries []yacymodel.RWIPosting,
) (Receipt, error) {
	if len(entries) > i.batchCap {
		return Receipt{Busy: true, TooLarge: true, Pause: i.pause}, nil
	}

	atCapacity, err := i.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return Receipt{Busy: true, Pause: i.pause}, nil
	}

	referenced := make([]yacymodel.URLHash, 0, len(entries))
	err = i.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}

			key := postingKey(entry.WordHash, entry.URLHash)
			if err := i.postings.Put(tx, key, entry); err != nil {
				return fmt.Errorf("store rwi posting: %w", err)
			}
			if err := i.observers.stored(tx, entry.WordHash, entry.URLHash); err != nil {
				return fmt.Errorf("note referenced url: %w", err)
			}
			referenced = append(referenced, entry.URLHash)
		}

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return Receipt{Busy: true, Pause: i.pause}, nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("store rwi: %w", err)
	}

	unknown, err := i.urls.MissingURLs(ctx, referenced)
	if err != nil {
		return Receipt{}, fmt.Errorf("missing urls: %w", err)
	}

	return Receipt{UnknownURL: unknown}, nil
}
