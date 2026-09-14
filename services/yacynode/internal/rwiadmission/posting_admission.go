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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type postingAdmission struct {
	vault      *vault.Vault
	urls       urlmeta.URLDirectory
	admitter   rwipostings.PostingAdmitter
	purger     rwipostings.PostingPurger
	references urlreferences.ReferenceQuery
	escrow     PostingHolder
	observer   RefusalObserver
	pause      time.Duration
}

func (a postingAdmission) Receive(
	ctx context.Context,
	entries []yacymodel.RWIPosting,
) (Receipt, error) {
	return a.receive(ctx, entries, keepEveryStoredPosting)
}

func keepEveryStoredPosting(*vault.Txn) error {
	return nil
}

func (a postingAdmission) ReceiveEveryPostingOfPage(
	ctx context.Context,
	pageURL yacymodel.URLHash,
	postings []yacymodel.RWIPosting,
) (Receipt, error) {
	return a.receive(ctx, postings, func(tx *vault.Txn) error {
		return a.purgeWordsDepartedFromPage(tx, pageURL, wordHashesOf(postings))
	})
}

func (a postingAdmission) purgeWordsDepartedFromPage(
	tx *vault.Txn,
	pageURL yacymodel.URLHash,
	pageWords map[yacymodel.Hash]struct{},
) error {
	storedWords, err := a.references.WordsReferencing(tx, pageURL)
	if err != nil {
		return fmt.Errorf("words referencing page: %w", err)
	}
	for _, word := range storedWords {
		if _, stays := pageWords[word]; stays {
			continue
		}
		if _, err := a.purger.PurgePosting(tx, word, pageURL); err != nil {
			return fmt.Errorf("purge posting of departed word: %w", err)
		}
	}

	return nil
}

func wordHashesOf(entries []yacymodel.RWIPosting) map[yacymodel.Hash]struct{} {
	hashes := make(map[yacymodel.Hash]struct{}, len(entries))
	for _, entry := range entries {
		hashes[entry.WordHash] = struct{}{}
	}

	return hashes
}

func (a postingAdmission) receive(
	ctx context.Context,
	entries []yacymodel.RWIPosting,
	purgeDepartedPostings func(tx *vault.Txn) error,
) (Receipt, error) {
	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		a.observer.ObserveRefused(RefusalStorageFull, len(entries))

		return Receipt{Busy: true, Pause: a.pause}, nil
	}

	referenced := urlHashesOf(entries)

	var unknown []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		missing, err := a.urls.MissingURLs(tx, referenced)
		if err != nil {
			return fmt.Errorf("missing urls: %w", err)
		}
		if err := purgeDepartedPostings(tx); err != nil {
			return err
		}
		if err := a.routeEach(ctx, tx, entries, awaitedURLsOf(missing)); err != nil {
			return err
		}
		unknown = missing

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

func urlHashesOf(entries []yacymodel.RWIPosting) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(entries))
	for _, entry := range entries {
		hashes = append(hashes, entry.URLHash)
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
	entries []yacymodel.RWIPosting,
	awaited map[yacymodel.URLHash]struct{},
) error {
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context: %w", err)
		}
		if err := a.route(tx, entry, awaited); err != nil {
			return err
		}
	}

	return nil
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

var (
	_ PostingReceiver     = postingAdmission{}
	_ PagePostingReceiver = postingAdmission{}
)
