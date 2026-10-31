package remoteindexadmission

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PostingOutcome string

const (
	PostingHeldForURLMetadata         PostingOutcome = "held_for_url_metadata"
	PostingDroppedFutureDate          PostingOutcome = "dropped_future_date"
	PostingDroppedStopword            PostingOutcome = "dropped_stopword"
	PostingRefusedTooManyPostings     PostingOutcome = "refused_too_many_postings"
	PostingRefusedStorageFull         PostingOutcome = "refused_storage_full"
	PostingRefusedEscrowFull          PostingOutcome = "refused_escrow_full"
	PostingIndexed                    PostingOutcome = "indexed"
	PostingDroppedAlreadyIndexed      PostingOutcome = "dropped_already_indexed"
	PostingDroppedNotNewerVersion     PostingOutcome = "dropped_not_newer_version"
	PostingDroppedPageFromOtherSource PostingOutcome = "dropped_page_from_other_source"
)

var errEscrowFull = errors.New("escrow holds as many postings as it can")

var _ PostingReceiver = PostingAdmission{}

type PostingAdmission struct {
	vault     *vault.Vault
	urls      urlmeta.URLDirectory
	escrow    Escrow
	placer    postingPlacer
	stopwords Stopwords
	observer  Observer
	now       func() time.Time

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
	outcome PostingOutcome,
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
	storedVersions, err := a.urls.VersionsOf(tx, referencedURLsOf(postings))
	if err != nil {
		return nil, fmt.Errorf("stored versions of urls: %w", err)
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
			storedVersionOf(storedVersions, posting.URLHash),
		)
		if err != nil {
			return nil, err
		}
		tx.RunAfterCommit(func() { a.observer.ObserveReceivedPostings(outcome, 1) })
		if outcome == PostingHeldForURLMetadata &&
			!slices.Contains(urlsOfHeldPostings, posting.URLHash) {
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
	storedVersions map[yacymodel.URLHash]pageversion.Version,
	url yacymodel.URLHash,
) yacymodel.Optional[pageversion.Version] {
	version, metadataStored := storedVersions[url]
	if !metadataStored {
		return yacymodel.None[pageversion.Version]()
	}

	return yacymodel.Some(version)
}

func (a PostingAdmission) admit(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedVersion yacymodel.Optional[pageversion.Version],
) (PostingOutcome, error) {
	arrivedVersion := pageversion.FromPosting(posting, pageversion.SourcePeer(origin))
	version, metadataStored := storedVersion.Get()
	switch {
	case a.stopwords.Contains(posting.WordHash, posting.Language):
		return PostingDroppedStopword, nil
	case arrivedVersion.IsDatedAfter(a.now()):
		return PostingDroppedFutureDate, nil
	case !metadataStored || version.IsReplacedBy(arrivedVersion):
		return a.hold(tx, origin, posting)
	}

	return a.placer.place(tx, posting, arrivedVersion, version)
}

func (a PostingAdmission) hold(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) (PostingOutcome, error) {
	if err := a.escrow.Hold(tx, origin, posting); err != nil {
		return "", fmt.Errorf("hold posting: %w", err)
	}

	return PostingHeldForURLMetadata, nil
}
