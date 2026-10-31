package remoteindexadmission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	PostingDroppedUnrootedVersion     PostingOutcome = "dropped_unrooted_version"
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
	receivedPagePostings []yacymodel.PagePostings,
) (PostingReceipt, error) {
	amountOfPostings := amountOfPostingsIn(receivedPagePostings)
	if amountOfPostings > a.postingCap {
		return a.busyReceiptRefusing(PostingRefusedTooManyPostings, amountOfPostings), nil
	}

	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return a.busyReceiptRefusing(PostingRefusedStorageFull, amountOfPostings), nil
	}
	escrowFull, err := a.escrow.IsFull(ctx)
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("check escrow capacity: %w", err)
	}

	var awaitedURLs []yacymodel.URLHash

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		urlsOfHeldPostings, err := a.admitEachPage(ctx, tx, origin, receivedPagePostings)
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
		return a.busyReceiptRefusing(PostingRefusedStorageFull, amountOfPostings), nil
	}
	if errors.Is(err, errEscrowFull) {
		return a.busyReceiptRefusing(PostingRefusedEscrowFull, amountOfPostings), nil
	}
	if err != nil {
		return PostingReceipt{}, fmt.Errorf("receive rwi: %w", err)
	}

	return PostingReceipt{UnknownURL: awaitedURLs}, nil
}

func amountOfPostingsIn(receivedPagePostings []yacymodel.PagePostings) int {
	amountOfPostings := 0
	for _, pagePostings := range receivedPagePostings {
		amountOfPostings += pagePostings.AmountOfPostings()
	}

	return amountOfPostings
}

func (a PostingAdmission) busyReceiptRefusing(
	outcome PostingOutcome,
	amountOfPostings int,
) PostingReceipt {
	a.observer.ObserveReceivedPostings(outcome, amountOfPostings)

	return PostingReceipt{Busy: true, Pause: a.pause}
}

func (a PostingAdmission) admitEachPage(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	receivedPagePostings []yacymodel.PagePostings,
) ([]yacymodel.URLHash, error) {
	storedVersions, err := a.urls.VersionsOf(tx, urlsOf(receivedPagePostings))
	if err != nil {
		return nil, fmt.Errorf("stored versions of urls: %w", err)
	}
	var urlsOfHeldPostings []yacymodel.URLHash
	for _, pagePostings := range receivedPagePostings {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context: %w", err)
		}
		url := pagePostings.URL()
		somePostingsHeld, err := a.admitPage(
			ctx,
			tx,
			origin,
			pagePostings,
			storedVersionOf(storedVersions, url),
		)
		if err != nil {
			return nil, err
		}
		if somePostingsHeld && !slices.Contains(urlsOfHeldPostings, url) {
			urlsOfHeldPostings = append(urlsOfHeldPostings, url)
		}
	}

	return urlsOfHeldPostings, nil
}

func urlsOf(receivedPagePostings []yacymodel.PagePostings) []yacymodel.URLHash {
	urls := make([]yacymodel.URLHash, 0, len(receivedPagePostings))
	for _, pagePostings := range receivedPagePostings {
		if !slices.Contains(urls, pagePostings.URL()) {
			urls = append(urls, pagePostings.URL())
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

func (a PostingAdmission) admitPage(
	ctx context.Context,
	tx *vault.Txn,
	origin rwiorigin.Origin,
	pagePostings yacymodel.PagePostings,
	storedVersion yacymodel.Optional[pageversion.Version],
) (somePostingsHeld bool, err error) {
	outcomes, err := a.admitEachPosting(tx, origin, pagePostings, storedVersion)
	if err != nil {
		return false, err
	}
	tx.RunAfterCommit(func() { a.observeReceived(outcomes) })
	passedPostings := passedPostingsOf(pagePostings.Postings(), outcomes)

	return slices.Contains(outcomes, PostingHeldForURLMetadata),
		a.indexPassedPostings(ctx, tx, pagePostings, passedPostings)
}

func (a PostingAdmission) admitEachPosting(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	pagePostings yacymodel.PagePostings,
	storedVersion yacymodel.Optional[pageversion.Version],
) ([]PostingOutcome, error) {
	if pagePostings.Proof().Present() {
		return a.admitProven(tx, origin, pagePostings.EachProvenPosting(), storedVersion)
	}

	return a.admitUnproven(tx, origin, pagePostings.Postings(), storedVersion)
}

func (a PostingAdmission) admitProven(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	provenPostings []yacymodel.ProvenPosting,
	storedVersion yacymodel.Optional[pageversion.Version],
) ([]PostingOutcome, error) {
	outcomes := make([]PostingOutcome, 0, len(provenPostings))
	for _, provenPosting := range provenPostings {
		arrivedVersion := pageversion.FromProvenPosting(
			provenPosting,
			pageversion.SourcePeer(origin),
		)
		outcome, err := a.admit(tx, provenPosting.Posting(), arrivedVersion, storedVersion)
		if err != nil {
			return nil, err
		}
		if outcome == PostingHeldForURLMetadata {
			if err := a.escrow.HoldProven(tx, origin, provenPosting, arrivedVersion); err != nil {
				return nil, fmt.Errorf("hold proven posting: %w", err)
			}
		}
		outcomes = append(outcomes, outcome)
	}

	return outcomes, nil
}

func (a PostingAdmission) admit(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	arrivedVersion pageversion.Version,
	storedVersion yacymodel.Optional[pageversion.Version],
) (PostingOutcome, error) {
	version, metadataStored := storedVersion.Get()
	switch {
	case a.stopwords.Contains(posting.WordHash, posting.Language):
		return PostingDroppedStopword, nil
	case arrivedVersion.IsDatedAfter(a.now()):
		return PostingDroppedFutureDate, nil
	case !metadataStored || version.IsReplacedBy(arrivedVersion):
		return PostingHeldForURLMetadata, nil
	}

	return a.placer.placementOf(tx, posting, arrivedVersion, version)
}

func (a PostingAdmission) admitUnproven(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	postings []yacymodel.RWIPosting,
	storedVersion yacymodel.Optional[pageversion.Version],
) ([]PostingOutcome, error) {
	outcomes := make([]PostingOutcome, 0, len(postings))
	for _, posting := range postings {
		arrivedVersion := pageversion.FromPosting(posting, pageversion.SourcePeer(origin))
		outcome, err := a.admit(tx, posting, arrivedVersion, storedVersion)
		if err != nil {
			return nil, err
		}
		if outcome == PostingHeldForURLMetadata {
			if err := a.escrow.HoldUnproven(tx, origin, posting, arrivedVersion); err != nil {
				return nil, fmt.Errorf("hold unproven posting: %w", err)
			}
		}
		outcomes = append(outcomes, outcome)
	}

	return outcomes, nil
}

func (a PostingAdmission) observeReceived(outcomes []PostingOutcome) {
	for _, outcome := range outcomes {
		a.observer.ObserveReceivedPostings(outcome, 1)
	}
}

func passedPostingsOf(
	postings []yacymodel.RWIPosting,
	outcomes []PostingOutcome,
) []yacymodel.RWIPosting {
	var passedPostings []yacymodel.RWIPosting
	for index, outcome := range outcomes {
		if outcome == PostingIndexed {
			passedPostings = append(passedPostings, postings[index])
		}
	}

	return passedPostings
}

func (a PostingAdmission) indexPassedPostings(
	ctx context.Context,
	tx *vault.Txn,
	pagePostings yacymodel.PagePostings,
	passedPostings []yacymodel.RWIPosting,
) error {
	if len(passedPostings) == 0 {
		return nil
	}
	passedPagePostings, present := pagePostings.NarrowedTo(passedPostings).Get()
	if !present {
		reportUnnarrowedPassedPostings(ctx, pagePostings.URL())

		return nil
	}

	return a.placer.index(tx, passedPagePostings)
}

func reportUnnarrowedPassedPostings(ctx context.Context, url yacymodel.URLHash) {
	slog.ErrorContext(ctx, "postings of a url passed admission outside their page postings",
		slog.String("url", url.String()),
	)
}
