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
	var postingsAloneToIndex []yacymodel.PostingAlone
	for _, waitingPosting := range waitingPostings {
		arrivedVersion := pageversion.FromPostingAlone(
			waitingPosting.PostingAlone(),
			pageversion.SourcePeer(waitingPosting.Origin()),
		)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		outcome, err := r.release(tx, waitingPosting, arrivedVersion, storedVersion)
		if err != nil {
			return err
		}
		if outcome == PostingIndexed {
			postingsAloneToIndex = append(postingsAloneToIndex, waitingPosting.PostingAlone())
		}
	}

	return r.indexReleased(ctx, tx, postingsAloneToIndex)
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

func (r waitingPostingReleaser) indexReleased(
	ctx context.Context,
	tx *vault.Txn,
	postingsAloneToIndex []yacymodel.PostingAlone,
) error {
	if len(postingsAloneToIndex) == 0 {
		return nil
	}
	released, joined := yacymodel.PagePostingsJoinedFrom(postingsAloneToIndex).Get()
	if !joined {
		reportUnjoinedReleasedPostings(ctx, postingsAloneToIndex[0].Posting().URLHash)

		return nil
	}

	return r.placer.index(tx, released)
}

func reportUnjoinedReleasedPostings(ctx context.Context, url yacymodel.URLHash) {
	slog.ErrorContext(ctx, "released postings of a url form no page postings",
		slog.String("url", url.String()),
	)
}
