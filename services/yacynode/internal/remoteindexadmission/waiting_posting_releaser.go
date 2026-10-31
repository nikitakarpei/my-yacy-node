package remoteindexadmission

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
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
	waitingPostings, err := r.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	var passedPostings []yacymodel.RWIPosting
	for _, waitingPosting := range waitingPostings {
		arrivedVersion := pageversion.FromPosting(
			waitingPosting.Posting,
			pageversion.SourcePeer(waitingPosting.Origin),
		)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		outcome, err := r.release(tx, waitingPosting.Posting, arrivedVersion, storedVersion)
		if err != nil {
			return err
		}
		if outcome == PostingIndexed {
			passedPostings = append(passedPostings, waitingPosting.Posting)
		}
	}

	return r.indexReleased(ctx, tx, passedPostings)
}

func (r waitingPostingReleaser) release(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	if err := r.escrow.Release(tx, posting); err != nil {
		return "", fmt.Errorf("release posting: %w", err)
	}
	outcome, err := r.placer.placementOf(tx, posting, arrivedVersion, storedVersion)
	if err != nil {
		return "", err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return outcome, nil
}

func (r waitingPostingReleaser) indexReleased(
	ctx context.Context,
	tx *vault.Txn,
	passedPostings []yacymodel.RWIPosting,
) error {
	if len(passedPostings) == 0 {
		return nil
	}
	released, err := yacymodel.UnprovenPagePostingsFrom(passedPostings)
	if err != nil {
		reportUnjoinedReleasedPostings(ctx, passedPostings[0].URLHash, err)

		return nil
	}

	return r.placer.index(tx, released)
}

func reportUnjoinedReleasedPostings(
	ctx context.Context,
	url yacymodel.URLHash,
	err error,
) {
	slog.ErrorContext(ctx, "released postings of a url form no page postings",
		slog.String("url", url.String()),
		slog.Any("error", err),
	)
}
