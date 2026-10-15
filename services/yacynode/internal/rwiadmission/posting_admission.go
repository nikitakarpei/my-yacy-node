package rwiadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

var errEscrowFull = errors.New("escrow holds as many postings as it can")

type postingAdmission struct {
	vault     *vault.Vault
	urls      urlmeta.URLDirectory
	versions  PageVersions
	escrow    PostingHolder
	stopwords Stopwords
	observer  RefusalObserver

	postingCap int
	pause      time.Duration
}

func (a postingAdmission) Receive(
	ctx context.Context,
	origin rwiorigin.Origin,
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

	escrowFull, err := a.escrow.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check escrow capacity: %w", err)
	}

	postingsWithoutStopwords := a.postingsWithoutStopwordsOf(postings)
	referenced := urlHashesOf(postingsWithoutStopwords)

	var unknown []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		metadata, err := a.urls.MetadataPerHash(tx, referenced)
		if err != nil {
			return fmt.Errorf("url metadata: %w", err)
		}
		heldURLs, err := a.routeEach(ctx, tx, origin, postingsWithoutStopwords, metadata)
		if err != nil {
			return err
		}
		if escrowFull && len(heldURLs) > 0 {
			return errEscrowFull
		}
		unknown = heldURLs

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		a.observer.ObserveRefused(RefusalStorageFull, len(postings))

		return a.busyReceipt(), nil
	}
	if errors.Is(err, errEscrowFull) {
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

func (a postingAdmission) postingsWithoutStopwordsOf(
	postings []yacymodel.RWIPosting,
) []yacymodel.RWIPosting {
	postingsWithoutStopwords := make([]yacymodel.RWIPosting, 0, len(postings))
	for _, posting := range postings {
		if !a.stopwords.Contains(posting.WordHash, posting.Language) {
			postingsWithoutStopwords = append(postingsWithoutStopwords, posting)
		}
	}

	return postingsWithoutStopwords
}

func urlHashesOf(postings []yacymodel.RWIPosting) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		hashes = append(hashes, posting.URLHash)
	}

	return hashes
}

func (a postingAdmission) routeEach(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata,
) ([]yacymodel.URLHash, error) {
	var heldURLs []yacymodel.URLHash
	seen := make(map[yacymodel.URLHash]struct{})
	for _, posting := range postings {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context: %w", err)
		}
		wasHeld, err := a.route(tx, origin, posting, metadata)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[posting.URLHash]; wasHeld && !ok {
			seen[posting.URLHash] = struct{}{}
			heldURLs = append(heldURLs, posting.URLHash)
		}
	}

	return heldURLs, nil
}

func (a postingAdmission) route(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata,
) (bool, error) {
	stored, known := metadata[posting.URLHash]
	if !known {
		return true, a.hold(tx, origin, posting)
	}
	verdict, err := a.versions.Admit(tx, origin, posting, stored.PageVersion())
	if err != nil {
		return false, fmt.Errorf("admit posting: %w", err)
	}
	if verdict.Outcome() != urlpageversion.OutcomeHeld {
		return false, nil
	}

	return true, a.hold(tx, origin, posting)
}

func (a postingAdmission) hold(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) error {
	if err := a.escrow.Hold(tx, origin, posting); err != nil {
		return fmt.Errorf("hold posting: %w", err)
	}

	return nil
}

var _ PostingReceiver = postingAdmission{}
