package remoteindexadmission

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
)

type waitingPostingReleaser struct {
	escrow   Escrow
	placer   postingPlacer
	observer Observer
}

func (r waitingPostingReleaser) releaseWaitingFor(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	storedVersion pageversion.Version,
) error {
	waitingPostings, err := r.escrow.PostingsWaitingFor(ctx, tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	passedProvenPostings, err := r.releaseProven(tx, waitingPostings.Proven, storedVersion)
	if err != nil {
		return err
	}
	passedPostings, err := r.releaseUnproven(tx, waitingPostings.Unproven, storedVersion)
	if err != nil {
		return err
	}

	return r.indexPassedPostings(ctx, tx, url, passedProvenPostings, passedPostings)
}

func (r waitingPostingReleaser) releaseProven(
	tx *vault.Txn,
	waitingPostings []rwiescrow.WaitingProvenPosting,
	storedVersion pageversion.Version,
) ([]yacymodel.ProvenPosting, error) {
	var passedPostings []yacymodel.ProvenPosting
	for _, waitingPosting := range waitingPostings {
		provenPosting := waitingPosting.ProvenPosting()
		source := pageversion.SourcePeer(waitingPosting.Origin())
		arrivedVersion := pageversion.FromProvenPosting(provenPosting, source)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		if err := r.escrow.ReleaseProven(tx, waitingPosting); err != nil {
			return nil, fmt.Errorf("release proven posting: %w", err)
		}
		outcome, err := r.observedOutcomeOf(
			tx,
			provenPosting.Posting(),
			arrivedVersion,
			storedVersion,
		)
		if err != nil {
			return nil, err
		}
		if outcome == PostingIndexed {
			passedPostings = append(passedPostings, provenPosting)
		}
	}

	return passedPostings, nil
}

func (r waitingPostingReleaser) observedOutcomeOf(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	outcome, err := r.placer.placementOf(tx, posting, arrivedVersion, storedVersion)
	if err != nil {
		return "", err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return outcome, nil
}

func (r waitingPostingReleaser) releaseUnproven(
	tx *vault.Txn,
	waitingPostings []rwiescrow.WaitingPosting,
	storedVersion pageversion.Version,
) ([]yacymodel.RWIPosting, error) {
	var passedPostings []yacymodel.RWIPosting
	for _, waitingPosting := range waitingPostings {
		posting := waitingPosting.Posting()
		source := pageversion.SourcePeer(waitingPosting.Origin())
		arrivedVersion := pageversion.FromPosting(posting, source)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		if err := r.escrow.ReleaseUnproven(tx, waitingPosting); err != nil {
			return nil, fmt.Errorf("release unproven posting: %w", err)
		}
		outcome, err := r.observedOutcomeOf(tx, posting, arrivedVersion, storedVersion)
		if err != nil {
			return nil, err
		}
		if outcome == PostingIndexed {
			passedPostings = append(passedPostings, posting)
		}
	}

	return passedPostings, nil
}

func (r waitingPostingReleaser) indexPassedPostings(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	passedProvenPostings []yacymodel.ProvenPosting,
	passedPostings []yacymodel.RWIPosting,
) error {
	if len(passedProvenPostings) == 0 && len(passedPostings) == 0 {
		return nil
	}
	passedPagePostings, err := pagePostingsFrom(passedProvenPostings, passedPostings)
	if err != nil {
		reportUnjoinedPassedPostings(ctx, url, err)

		return nil
	}

	return r.placer.index(tx, passedPagePostings)
}

func pagePostingsFrom(
	passedProvenPostings []yacymodel.ProvenPosting,
	passedPostings []yacymodel.RWIPosting,
) (yacymodel.PagePostings, error) {
	if len(passedProvenPostings) > 0 {
		return yacymodel.PagePostingsJoinedFrom(passedProvenPostings)
	}

	return yacymodel.UnprovenPagePostingsFrom(passedPostings)
}

func reportUnjoinedPassedPostings(ctx context.Context, url yacymodel.URLHash, err error) {
	slog.ErrorContext(ctx, "postings of a url passed release and form no page postings",
		slog.String("url", url.String()),
		slog.Any("error", err),
	)
}
