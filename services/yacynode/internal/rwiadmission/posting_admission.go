package rwiadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type postingAdmission struct {
	vault    *vault.Vault
	urls     urlmeta.URLDirectory
	admitter rwipostings.PostingAdmitter
	escrow   PostingHolder
	observer RefusalObserver
	batchCap int
	pause    time.Duration
}

func (a postingAdmission) Receive(
	ctx context.Context,
	entries []yacymodel.RWIPosting,
) (Receipt, error) {
	if len(entries) > a.batchCap {
		a.observer.ObserveRefused(RefusalRequestTooLarge, len(entries))

		return Receipt{Busy: true, TooLarge: true, Pause: a.pause}, nil
	}

	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		a.observer.ObserveRefused(RefusalStorageFull, len(entries))

		return Receipt{Busy: true, Pause: a.pause}, nil
	}

	referenced := make([]yacymodel.URLHash, 0, len(entries))
	for _, entry := range entries {
		referenced = append(referenced, entry.URLHash)
	}
	unknown, err := a.urls.MissingURLs(ctx, referenced)
	if err != nil {
		return Receipt{}, fmt.Errorf("missing urls: %w", err)
	}

	awaited := make(map[yacymodel.URLHash]struct{}, len(unknown))
	for _, hash := range unknown {
		awaited[hash] = struct{}{}
	}

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("context: %w", err)
			}
			if err := a.route(tx, entry, awaited); err != nil {
				return err
			}
		}

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		a.observer.ObserveRefused(RefusalStorageFull, len(entries))

		return Receipt{Busy: true, Pause: a.pause}, nil
	}
	if errors.Is(err, rwiescrow.ErrEscrowFull) {
		a.observer.ObserveRefused(RefusalEscrowFull, len(entries))

		return Receipt{Busy: true, Pause: a.pause}, nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("receive rwi: %w", err)
	}

	return Receipt{UnknownURL: unknown}, nil
}

func (a postingAdmission) route(
	tx *vault.Txn,
	entry yacymodel.RWIPosting,
	awaited map[yacymodel.URLHash]struct{},
) error {
	if _, waits := awaited[entry.URLHash]; waits {
		if err := a.escrow.Hold(tx, entry); err != nil {
			return fmt.Errorf("hold posting: %w", err)
		}

		return nil
	}
	if err := a.admitter.Admit(tx, entry); err != nil {
		return fmt.Errorf("admit posting: %w", err)
	}

	return nil
}

var _ PostingReceiver = postingAdmission{}
