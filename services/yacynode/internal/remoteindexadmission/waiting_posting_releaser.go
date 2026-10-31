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
	var passedPostings []rwiescrow.WaitingPosting
	for _, waitingPosting := range waitingPostings {
		arrivedVersion := arrivedVersionOf(waitingPosting)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		outcome, err := r.release(tx, waitingPosting, arrivedVersion, storedVersion)
		if err != nil {
			return err
		}
		if outcome == PostingIndexed {
			passedPostings = append(passedPostings, waitingPosting)
		}
	}

	return r.indexPassedPostings(ctx, tx, url, passedPostings)
}

func arrivedVersionOf(waitingPosting rwiescrow.WaitingPosting) pageversion.Version {
	source := pageversion.SourcePeer(waitingPosting.Origin())
	provenPosting, proven := waitingPosting.ProvenPosting().Get()
	if proven {
		return pageversion.FromProvenPosting(provenPosting, source)
	}

	return pageversion.FromPosting(waitingPosting.Posting(), source)
}

func (r waitingPostingReleaser) release(
	tx *vault.Txn,
	waitingPosting rwiescrow.WaitingPosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	if err := r.escrow.Release(tx, waitingPosting); err != nil {
		return "", fmt.Errorf("release posting: %w", err)
	}
	outcome, err := r.placer.placementOf(
		tx,
		waitingPosting.Posting(),
		arrivedVersion,
		storedVersion,
	)
	if err != nil {
		return "", err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return outcome, nil
}

func (r waitingPostingReleaser) indexPassedPostings(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	passedPostings []rwiescrow.WaitingPosting,
) error {
	if len(passedPostings) == 0 {
		return nil
	}
	passedPagePostings, err := pagePostingsFrom(passedPostings)
	if err != nil {
		reportUnjoinedPassedPostings(ctx, url, err)

		return nil
	}

	return r.placer.index(tx, passedPagePostings)
}

func pagePostingsFrom(
	passedPostings []rwiescrow.WaitingPosting,
) (yacymodel.PagePostings, error) {
	postings := make([]yacymodel.RWIPosting, 0, len(passedPostings))
	provenPostings := make([]yacymodel.ProvenPosting, 0, len(passedPostings))
	for _, passedPosting := range passedPostings {
		postings = append(postings, passedPosting.Posting())
		if provenPosting, proven := passedPosting.ProvenPosting().Get(); proven {
			provenPostings = append(provenPostings, provenPosting)
		}
	}
	if len(provenPostings) > 0 {
		return yacymodel.PagePostingsJoinedFrom(provenPostings)
	}

	return yacymodel.UnprovenPagePostingsFrom(postings)
}

func reportUnjoinedPassedPostings(ctx context.Context, url yacymodel.URLHash, err error) {
	slog.ErrorContext(ctx, "postings of a url passed release and form no page postings",
		slog.String("url", url.String()),
		slog.Any("error", err),
	)
}
