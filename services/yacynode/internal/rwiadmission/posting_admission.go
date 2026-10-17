package rwiadmission

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiindexedversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var errEscrowFull = errors.New("escrow holds as many postings as it can")

var _ PostingReceiver = (*PostingAdmission)(nil)

type PostingAdmission struct {
	vault           *vault.Vault
	urls            urlmeta.URLDirectory
	escrow          Escrow
	judge           postingJudge
	indexedVersions IndexedVersions
	references      rwipostings.URLReferences
	admitter        rwipostings.PostingAdmitter
	purger          rwipostings.PostingPurger
	swaps           *swapAllowance
	stopwords       Stopwords
	observer        Observer

	postingCap int
	pause      time.Duration
}

func (a *PostingAdmission) Receive(
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
		awaitedURLs, err := a.placeEach(ctx, tx, origin, postingsWithoutStopwords, metadata)
		if err != nil {
			return err
		}
		if escrowFull && len(awaitedURLs) > 0 {
			return errEscrowFull
		}
		unknown = awaitedURLs

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

func (a *PostingAdmission) busyReceipt() Receipt {
	return Receipt{Busy: true, Pause: a.pause}
}

func (a *PostingAdmission) postingsWithoutStopwordsOf(
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

func (a *PostingAdmission) placeEach(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata,
) ([]yacymodel.URLHash, error) {
	var awaitedURLs []yacymodel.URLHash
	for _, posting := range postings {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context: %w", err)
		}
		wasHeld, err := a.placeReceived(
			tx,
			origin,
			posting,
			storedFingerprintOf(metadata, posting.URLHash),
		)
		if err != nil {
			return nil, err
		}
		if wasHeld && !slices.Contains(awaitedURLs, posting.URLHash) {
			awaitedURLs = append(awaitedURLs, posting.URLHash)
		}
	}

	return awaitedURLs, nil
}

func storedFingerprintOf(
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata,
	url yacymodel.URLHash,
) yacymodel.Optional[pagefingerprint.Fingerprint] {
	stored, known := metadata[url]
	if !known {
		return yacymodel.None[pagefingerprint.Fingerprint]()
	}

	return yacymodel.Some(pagefingerprint.OfMetadata(stored))
}

func (a *PostingAdmission) placeReceived(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedFingerprint yacymodel.Optional[pagefingerprint.Fingerprint],
) (bool, error) {
	outcome, err := a.judge.outcomeOf(tx, origin, posting, storedFingerprint)
	if err != nil {
		return false, err
	}
	a.observer.ObserveOutcome(tx, outcome)
	if outcome == OutcomeHeldInEscrow {
		if err := a.escrow.Hold(tx, origin, posting); err != nil {
			return false, fmt.Errorf("hold posting: %w", err)
		}

		return true, nil
	}

	return false, a.carryOut(tx, origin, posting, outcome)
}

func (a *PostingAdmission) carryOut(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	outcome Outcome,
) error {
	switch outcome {
	case OutcomeIndexedFirstVersion:
		return a.indexFirstVersion(tx, origin, posting)
	case OutcomeIndexedMissingWord:
		return a.indexMissingWord(tx, posting)
	case OutcomeIndexedNewerVersion:
		return a.indexNewerVersion(tx, origin, posting)
	default:
		return nil
	}
}

func (a *PostingAdmission) indexFirstVersion(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) error {
	if err := a.admit(tx, posting); err != nil {
		return err
	}

	return a.indexedVersions.Record(tx, posting.URLHash, rwiindexedversion.IndexedVersion{
		Fingerprint: pagefingerprint.OfPosting(posting),
		Origin:      origin,
	})
}

func (a *PostingAdmission) admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := a.admitter.Admit(tx, posting); err != nil {
		return fmt.Errorf("admit posting: %w", err)
	}

	return nil
}

func (a *PostingAdmission) indexMissingWord(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	return a.admit(tx, posting)
}

func (a *PostingAdmission) indexNewerVersion(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) error {
	tx.RunAfterCommit(func() { a.swaps.count(origin.Network) })
	if err := a.purgePostingsOf(tx, posting.URLHash); err != nil {
		return err
	}

	return a.indexFirstVersion(tx, origin, posting)
}

func (a *PostingAdmission) purgePostingsOf(tx *vault.Txn, url yacymodel.URLHash) error {
	words, err := a.references.WordsReferencing(tx, url)
	if err != nil {
		return fmt.Errorf("words of the replaced page version: %w", err)
	}
	for _, word := range words {
		if _, err := a.purger.PurgePosting(tx, word, url); err != nil {
			return fmt.Errorf("purge posting of the replaced page version: %w", err)
		}
	}

	return nil
}

func (a *PostingAdmission) URLStored(
	tx *vault.Txn,
	url yacymodel.URLHash,
	storedFingerprint pagefingerprint.Fingerprint,
) error {
	waiting, err := a.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	for _, posting := range waiting {
		if err := a.placeWaiting(tx, posting, storedFingerprint); err != nil {
			return err
		}
	}

	return nil
}

func (a *PostingAdmission) placeWaiting(
	tx *vault.Txn,
	waiting rwiescrow.WaitingPosting,
	storedFingerprint pagefingerprint.Fingerprint,
) error {
	outcome, err := a.judge.outcomeOf(
		tx,
		waiting.Origin,
		waiting.Posting,
		yacymodel.Some(storedFingerprint),
	)
	if err != nil {
		return err
	}
	if outcome == OutcomeHeldInEscrow {
		return nil
	}
	a.observer.ObserveOutcome(tx, outcome)
	if err := a.carryOut(tx, waiting.Origin, waiting.Posting, outcome); err != nil {
		return err
	}
	if err := a.escrow.Release(tx, waiting.Posting); err != nil {
		return fmt.Errorf("release posting: %w", err)
	}

	return nil
}
