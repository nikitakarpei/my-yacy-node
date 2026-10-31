package remoteindexadmission

import (
	"fmt"

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
	tx *vault.Txn,
	url yacymodel.URLHash,
	storedVersion pageversion.Version,
) error {
	waitingPostings, err := r.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	var postingsToIndex []yacymodel.RWIPosting
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
			postingsToIndex = append(postingsToIndex, waitingPosting.Posting)
		}
	}

	return r.indexReleased(tx, postingsToIndex)
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
	tx *vault.Txn,
	postingsToIndex []yacymodel.RWIPosting,
) error {
	if len(postingsToIndex) == 0 {
		return nil
	}
	released, err := yacymodel.UnprovenPagePostingsFrom(postingsToIndex)
	if err != nil {
		return fmt.Errorf("released postings: %w", err)
	}

	return r.placer.index(tx, released, postingsToIndex)
}
