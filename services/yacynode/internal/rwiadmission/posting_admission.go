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

	postingCap int
	pause      time.Duration
}

func (a postingAdmission) Receive(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
) (Receipt, error) {
	if len(postings) > a.postingCap {
		a.observer.ObserveRefused(RefusalTooManyPostings, len(postings))

		return a.busyReceipt(), nil
	}

	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		a.observer.ObserveRefused(RefusalStorageFull, len(postings))

		return a.busyReceipt(), nil
	}

	referenced := urlHashesOf(postings)

	var unknown []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		missing, err := a.urls.MissingURLs(tx, referenced)
		if err != nil {
			return fmt.Errorf("missing urls: %w", err)
		}
		if err := a.routeEach(ctx, tx, postings, awaitedURLsOf(missing)); err != nil {
			return err
		}
		unknown = missing

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		a.observer.ObserveRefused(RefusalStorageFull, len(postings))

		return a.busyReceipt(), nil
	}
	if errors.Is(err, rwiescrow.ErrEscrowFull) {
		a.observer.ObserveRefused(RefusalEscrowFull, len(postings))

		return a.busyReceipt(), nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("receive rwi: %w", err)
	}

	return Receipt{UnknownURL: unknown}, nil
}

func (a postingAdmission) busyReceipt() Receipt {
	return Receipt{Busy: true, Pause: a.pause}
}

func urlHashesOf(postings []yacymodel.RWIPosting) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		hashes = append(hashes, posting.URLHash)
	}

	return hashes
}

func awaitedURLsOf(missing []yacymodel.URLHash) map[yacymodel.URLHash]struct{} {
	awaited := make(map[yacymodel.URLHash]struct{}, len(missing))
	for _, hash := range missing {
		awaited[hash] = struct{}{}
	}

	return awaited
}

func (a postingAdmission) routeEach(
	ctx context.Context,
	tx *vault.Txn,
	postings []yacymodel.RWIPosting,
	awaited map[yacymodel.URLHash]struct{},
) error {
	for _, posting := range postings {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context: %w", err)
		}
		if err := a.route(tx, posting, awaited); err != nil {
			return err
		}
	}

	return nil
}

func (a postingAdmission) route(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	awaited map[yacymodel.URLHash]struct{},
) error {
	if _, waits := awaited[posting.URLHash]; waits {
		if err := a.escrow.Hold(tx, posting); err != nil {
			return fmt.Errorf("hold posting: %w", err)
		}

		return nil
	}
	if err := a.admitter.Admit(tx, posting); err != nil {
		return fmt.Errorf("admit posting: %w", err)
	}

	return nil
}

var _ PostingReceiver = postingAdmission{}
