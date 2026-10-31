package rwiescrow

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type postingIdentity struct {
	Word    yacymodel.Hash
	URL     yacymodel.URLHash
	Version []byte
}

type escrowedPosting struct {
	HeldAt  time.Time
	Origin  rwiorigin.Origin
	Posting yacymodel.RWIPosting
	Proof   yacymodel.Optional[yacymodel.PostingsProof]
}

type postingHold struct {
	HeldAt  time.Time
	Posting postingIdentity
}

type PostingEscrow struct {
	vault    *vault.Vault
	escrowed *vault.Collection[postingIdentity, escrowedPosting]
	holds    *vault.ValueIndex[postingHold, postingHold]
	capacity int
	now      func() time.Time
}

func (e *PostingEscrow) Hold(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	pagePosting yacymodel.PagePosting,
	version pageversion.Version,
) error {
	posting := pagePosting.Posting()
	identity := postingIdentity{
		Word:    posting.WordHash,
		URL:     posting.URLHash,
		Version: pageversion.AppendBytes(version, nil),
	}
	_, err := e.escrowed.Put(tx, identity, escrowedPosting{
		HeldAt:  e.now(),
		Origin:  origin,
		Posting: posting,
		Proof:   pagePosting.Proof(),
	})
	if err != nil {
		return fmt.Errorf("escrow posting: %w", err)
	}

	return nil
}

func (e *PostingEscrow) PostingsWaitingFor(
	ctx context.Context,
	tx *vault.Txn,
	url yacymodel.URLHash,
) ([]WaitingPosting, error) {
	var waitingPostings []WaitingPosting
	err := e.escrowed.Scan(
		tx,
		everyPostingWaitingFor(url),
		func(identity postingIdentity, escrowed escrowedPosting) (bool, error) {
			waitingPosting, err := waitingPostingFrom(identity, escrowed)
			if err != nil {
				reportRefusedPosting(ctx, identity, err)

				return true, nil
			}
			waitingPostings = append(waitingPostings, waitingPosting)

			return true, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read postings waiting for url: %w", err)
	}

	return waitingPostings, nil
}

func reportRefusedPosting(ctx context.Context, identity postingIdentity, err error) {
	slog.WarnContext(ctx, "escrowed posting refused, posting left to expire",
		slog.String("url", identity.URL.String()),
		slog.String("word", identity.Word.String()),
		slog.Any("error", err),
	)
}

func (e *PostingEscrow) Release(tx *vault.Txn, waiting WaitingPosting) error {
	return e.drop(tx, waiting.identity)
}

func (e *PostingEscrow) Expire(
	ctx context.Context,
	holdFor time.Duration,
	limit int,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	cutoff := e.now().Add(-holdFor)
	var expired int
	err := e.vault.Update(ctx, func(tx *vault.Txn) error {
		expiredHolds, err := e.postingHoldsBefore(tx, cutoff, limit)
		if err != nil {
			return err
		}
		for _, hold := range expiredHolds {
			if err := e.drop(tx, hold.Posting); err != nil {
				return err
			}
		}
		expired = len(expiredHolds)

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("expire escrowed postings: %w", err)
	}

	return expired, nil
}

func (e *PostingEscrow) IsFull(ctx context.Context) (bool, error) {
	var held int
	err := e.vault.View(ctx, func(tx *vault.Txn) error {
		count, err := e.Count(tx)
		held = count

		return err
	})
	if err != nil {
		return false, err
	}

	return held >= e.capacity, nil
}

func (e *PostingEscrow) Count(tx *vault.Txn) (int, error) {
	count, err := e.escrowed.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("read escrowed posting length: %w", err)
	}

	return count, nil
}

func (e *PostingEscrow) Capacity() int {
	return e.capacity
}

func (e *PostingEscrow) postingHoldsBefore(
	tx *vault.Txn,
	cutoff time.Time,
	limit int,
) ([]postingHold, error) {
	holds := make([]postingHold, 0, limit)
	err := e.holds.Scan(tx, everyHoldPlacedBefore(cutoff), func(hold postingHold) (bool, error) {
		holds = append(holds, hold)

		return len(holds) < limit, nil
	})
	if err != nil {
		return nil, fmt.Errorf("read posting holds placed before the cutoff: %w", err)
	}

	return holds, nil
}

func (e *PostingEscrow) drop(tx *vault.Txn, identity postingIdentity) error {
	if _, err := e.escrowed.Delete(tx, identity); err != nil {
		return fmt.Errorf("drop escrowed posting: %w", err)
	}

	return nil
}
