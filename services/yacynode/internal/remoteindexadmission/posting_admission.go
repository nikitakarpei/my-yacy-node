package remoteindexadmission

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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type ReceivedPostingOutcome string

const (
	PostingHeldInEscrow           ReceivedPostingOutcome = "held_in_escrow"
	PostingDroppedFutureDate      ReceivedPostingOutcome = "dropped_future_date"
	PostingDroppedStopword        ReceivedPostingOutcome = "dropped_stopword"
	PostingRefusedTooManyPostings ReceivedPostingOutcome = "refused_too_many_postings"
	PostingRefusedStorageFull     ReceivedPostingOutcome = "refused_storage_full"
	PostingRefusedEscrowFull      ReceivedPostingOutcome = "refused_escrow_full"
)

type PostingPlacement string

const (
	PostingIndexed               PostingPlacement = "indexed"
	PostingDroppedAlreadyIndexed PostingPlacement = "dropped_already_indexed"
	PostingDroppedOtherVersion   PostingPlacement = "dropped_other_version"
	PostingDroppedOtherNetwork   PostingPlacement = "dropped_other_network"
)

var errEscrowFull = errors.New("escrow holds as many postings as it can")

var _ PostingReceiver = PostingAdmission{}

type PostingAdmission struct {
	vault         *vault.Vault
	urls          urlmeta.URLDirectory
	escrow        Escrow
	postings      rwipostings.PostingIndex
	postingWriter rwipostings.PostingWriter
	stopwords     Stopwords
	observer      Observer
	now           func() time.Time

	postingCap int
	pause      time.Duration
}

func (a PostingAdmission) Receive(
	ctx context.Context,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
) (PostingReceipt, error) {
	if len(postings) > a.postingCap {
		return a.busyReceiptRefusing(PostingRefusedTooManyPostings, postings), nil
	}

	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return a.busyReceiptRefusing(PostingRefusedStorageFull, postings), nil
	}
	escrowFull, err := a.escrow.IsFull(ctx)
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("check escrow capacity: %w", err)
	}

	var awaitedURLs []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		urlsOfHeldPostings, err := a.admitEach(ctx, tx, origin, postings)
		if err != nil {
			return err
		}
		if escrowFull && len(urlsOfHeldPostings) > 0 {
			return errEscrowFull
		}
		awaitedURLs = urlsOfHeldPostings

		return nil
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return a.busyReceiptRefusing(PostingRefusedStorageFull, postings), nil
	}
	if errors.Is(err, errEscrowFull) {
		return a.busyReceiptRefusing(PostingRefusedEscrowFull, postings), nil
	}
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("receive rwi: %w", err)
	}

	return PostingReceipt{UnknownURL: awaitedURLs}, nil
}

func (a PostingAdmission) busyReceiptRefusing(
	outcome ReceivedPostingOutcome,
	postings []yacymodel.RWIPosting,
) PostingReceipt {
	a.observer.ObserveReceivedPostings(outcome, len(postings))

	return PostingReceipt{Busy: true, Pause: a.pause}
}

func (a PostingAdmission) admitEach(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
) ([]yacymodel.URLHash, error) {
	currentVersions, err := a.urls.CurrentVersionsOf(tx, referencedURLsOf(postings))
	if err != nil {
		return nil, fmt.Errorf("current versions of urls: %w", err)
	}
	var urlsOfHeldPostings []yacymodel.URLHash
	for _, posting := range postings {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context: %w", err)
		}
		outcome, err := a.admit(
			tx,
			origin,
			posting,
			storedVersionOf(currentVersions, posting.URLHash),
		)
		if err != nil {
			return nil, err
		}
		tx.RunAfterCommit(func() { a.observer.ObserveReceivedPostings(outcome, 1) })
		if outcome == PostingHeldInEscrow && !slices.Contains(urlsOfHeldPostings, posting.URLHash) {
			urlsOfHeldPostings = append(urlsOfHeldPostings, posting.URLHash)
		}
	}

	return urlsOfHeldPostings, nil
}

func referencedURLsOf(postings []yacymodel.RWIPosting) []yacymodel.URLHash {
	urls := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		if !slices.Contains(urls, posting.URLHash) {
			urls = append(urls, posting.URLHash)
		}
	}

	return urls
}

func storedVersionOf(
	currentVersions map[yacymodel.URLHash]urlmeta.CurrentVersion,
	url yacymodel.URLHash,
) yacymodel.Optional[urlmeta.CurrentVersion] {
	version, found := currentVersions[url]
	if !found {
		return yacymodel.None[urlmeta.CurrentVersion]()
	}

	return yacymodel.Some(version)
}

func (a PostingAdmission) admit(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedVersion yacymodel.Optional[urlmeta.CurrentVersion],
) (ReceivedPostingOutcome, error) {
	postingFingerprint := pagefingerprint.FromPosting(posting)
	currentVersion, metadataStored := storedVersion.Get()
	switch {
	case a.stopwords.Contains(posting.WordHash, posting.Language):
		return PostingDroppedStopword, nil
	case postingFingerprint.IsDatedAfter(a.now()):
		return PostingDroppedFutureDate, nil
	case !metadataStored || currentVersion.IsOlderThan(postingFingerprint):
		return a.hold(tx, origin, posting)
	}
	placement, err := a.place(tx, origin, posting, currentVersion)

	return ReceivedPostingOutcome(placement), err
}

func (a PostingAdmission) hold(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) (ReceivedPostingOutcome, error) {
	if err := a.escrow.Hold(tx, origin, posting); err != nil {
		return "", fmt.Errorf("hold posting: %w", err)
	}

	return PostingHeldInEscrow, nil
}

func (a PostingAdmission) place(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	currentVersion urlmeta.CurrentVersion,
) (PostingPlacement, error) {
	switch {
	case !currentVersion.Describes(pagefingerprint.FromPosting(posting)):
		return PostingDroppedOtherVersion, nil
	case !currentVersion.CameFrom(origin):
		return PostingDroppedOtherNetwork, nil
	}
	postingIndexed, err := a.isIndexed(tx, posting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return PostingDroppedAlreadyIndexed, nil
	}

	return a.write(tx, posting)
}

func (a PostingAdmission) isIndexed(tx *vault.Txn, posting yacymodel.RWIPosting) (bool, error) {
	_, indexed, err := a.postings.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}

func (a PostingAdmission) write(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) (PostingPlacement, error) {
	if err := a.postingWriter.Write(tx, posting); err != nil {
		return "", fmt.Errorf("write posting: %w", err)
	}

	return PostingIndexed, nil
}

func (a PostingAdmission) releaseWaitingFor(
	tx *vault.Txn,
	url yacymodel.URLHash,
	currentVersion urlmeta.CurrentVersion,
) error {
	waiting, err := a.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	for _, posting := range waiting {
		if currentVersion.IsOlderThan(pagefingerprint.FromPosting(posting.Posting)) {
			continue
		}
		if err := a.release(tx, posting, currentVersion); err != nil {
			return err
		}
	}

	return nil
}

func (a PostingAdmission) release(
	tx *vault.Txn,
	waiting rwiescrow.WaitingPosting,
	currentVersion urlmeta.CurrentVersion,
) error {
	if err := a.escrow.Release(tx, waiting.Posting); err != nil {
		return fmt.Errorf("release posting: %w", err)
	}
	placement, err := a.place(tx, waiting.Origin, waiting.Posting, currentVersion)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() { a.observer.ObserveReleasedPostings(placement, 1) })

	return nil
}
