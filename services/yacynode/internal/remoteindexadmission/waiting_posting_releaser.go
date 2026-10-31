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

	return r.indexReleased(ctx, tx, url, passedPostings)
}

func arrivedVersionOf(waitingPosting rwiescrow.WaitingPosting) pageversion.Version {
	source := pageversion.SourcePeer(waitingPosting.Origin())
	provenPosting, proven := waitingPosting.Proven().Get()
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

func (r waitingPostingReleaser) indexReleased(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	releasedPostings []rwiescrow.WaitingPosting,
) error {
	provenPostings, unprovenPostings := provenAndUnprovenOf(releasedPostings)
	if err := r.indexReleasedProven(ctx, tx, url, provenPostings); err != nil {
		return err
	}

	return r.indexReleasedUnproven(ctx, tx, url, unprovenPostings)
}

func provenAndUnprovenOf(
	releasedPostings []rwiescrow.WaitingPosting,
) ([]yacymodel.ProvenPosting, []yacymodel.RWIPosting) {
	var provenPostings []yacymodel.ProvenPosting
	var unprovenPostings []yacymodel.RWIPosting
	for _, releasedPosting := range releasedPostings {
		provenPosting, proven := releasedPosting.Proven().Get()
		if proven {
			provenPostings = append(provenPostings, provenPosting)
		} else {
			unprovenPostings = append(unprovenPostings, releasedPosting.Posting())
		}
	}

	return provenPostings, unprovenPostings
}

func (r waitingPostingReleaser) indexReleasedProven(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	provenPostings []yacymodel.ProvenPosting,
) error {
	if len(provenPostings) == 0 {
		return nil
	}
	released, joined := yacymodel.ProvenPagePostingsJoinedFrom(provenPostings).Get()
	if !joined {
		reportUnjoinedReleasedPostings(ctx, url)

		return nil
	}

	return r.placer.index(tx, released)
}

func reportUnjoinedReleasedPostings(ctx context.Context, url yacymodel.URLHash) {
	slog.ErrorContext(ctx, "released postings of a url form no page postings",
		slog.String("url", url.String()),
	)
}

func (r waitingPostingReleaser) indexReleasedUnproven(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
	unprovenPostings []yacymodel.RWIPosting,
) error {
	if len(unprovenPostings) == 0 {
		return nil
	}
	released, err := yacymodel.UnprovenPagePostingsFrom(unprovenPostings)
	if err != nil {
		reportUnjoinedReleasedPostings(ctx, url)

		return nil //nolint:nilerr
	}

	return r.placer.index(tx, released)
}
