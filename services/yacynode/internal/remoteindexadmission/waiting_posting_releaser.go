package remoteindexadmission

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type waitingPostingReleaser struct {
	escrow   Escrow
	placer   postingPlacer
	observer Observer
}

func (r waitingPostingReleaser) releaseWaitingFor(
	tx *vault.Txn,
	url yacymodel.URLHash,
	currentVersion urlmeta.CurrentVersion,
) error {
	waitingPostings, err := r.escrow.PostingsWaitingFor(tx, url)
	if err != nil {
		return fmt.Errorf("postings waiting for url: %w", err)
	}
	for _, waitingPosting := range waitingPostings {
		if currentVersion.IsOlderThan(pagefingerprint.FromPosting(waitingPosting.Posting)) {
			continue
		}
		if err := r.placeReleased(tx, waitingPosting, currentVersion); err != nil {
			return err
		}
	}

	return nil
}

func (r waitingPostingReleaser) placeReleased(
	tx *vault.Txn,
	waitingPosting rwiescrow.WaitingPosting,
	currentVersion urlmeta.CurrentVersion,
) error {
	if err := r.escrow.Release(tx, waitingPosting.Posting); err != nil {
		return fmt.Errorf("release posting: %w", err)
	}
	outcome, err := r.placer.place(
		tx,
		waitingPosting.Origin,
		waitingPosting.Posting,
		currentVersion,
	)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() { r.observer.ObserveReleasedPostings(outcome, 1) })

	return nil
}
