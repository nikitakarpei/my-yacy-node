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
	for _, waitingPosting := range waitingPostings {
		arrivedVersion := pageversion.FromPosting(
			waitingPosting.Posting,
			pageversion.SourcePeer(waitingPosting.Origin),
		)
		if storedVersion.IsReplacedBy(arrivedVersion) {
			continue
		}
		if err := r.placeReleased(
			tx,
			waitingPosting.Posting,
			arrivedVersion,
			storedVersion,
		); err != nil {
			return err
		}
	}

	return nil
}

func (r waitingPostingReleaser) placeReleased(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) error {
	if err := r.escrow.Release(tx, posting); err != nil {
		return fmt.Errorf("release posting: %w", err)
	}
	outcome, err := r.placer.place(tx, posting, arrivedVersion, storedVersion)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return nil
}
