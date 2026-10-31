package remoteindexadmission

import (
	"context"
	"fmt"

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
	passedPagePostings, err := r.release(tx, waitingPostings, storedVersion)
	if err != nil {
		return err
	}

	return r.placer.indexJoined(ctx, tx, passedPagePostings)
}

func (r waitingPostingReleaser) release(
	tx *vault.Txn,
	waitingPostings []rwiescrow.WaitingPosting,
	storedVersion pageversion.Version,
) ([]yacymodel.PagePosting, error) {
	var passedPagePostings []yacymodel.PagePosting
	for _, waitingPosting := range waitingPostings {
		if storedVersion.IsReplacedBy(waitingPosting.Version()) {
			continue
		}
		if err := r.escrow.Release(tx, waitingPosting); err != nil {
			return nil, fmt.Errorf("release posting: %w", err)
		}
		outcome, err := r.observedOutcomeOf(tx, waitingPosting, storedVersion)
		if err != nil {
			return nil, err
		}
		if outcome == PostingIndexed {
			passedPagePostings = append(passedPagePostings, waitingPosting.PagePosting())
		}
	}

	return passedPagePostings, nil
}

func (r waitingPostingReleaser) observedOutcomeOf(
	tx *vault.Txn,
	waitingPosting rwiescrow.WaitingPosting,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	outcome, err := r.placer.placementOf(
		tx,
		waitingPosting.PagePosting(),
		waitingPosting.Version(),
		storedVersion,
	)
	if err != nil {
		return "", err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return outcome, nil
}
