package rwiescrow

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type escrowedPosting struct {
	HeldAt  time.Time
	Origin  rwiorigin.Origin
	Posting yacymodel.RWIPosting
}

type WaitingPosting struct {
	Origin  rwiorigin.Origin
	Posting yacymodel.RWIPosting
}

type postingHold struct {
	HeldAt  time.Time
	Posting postingIdentity
}

type PostingEscrow struct {
	vault    *vault.Vault
	escrowed *vault.Collection[postingIdentity, escrowedPosting]
	holds    *vault.ValueIndex[postingHold, postingHold]
	observer HoldObserver
	capacity int
	now      func() time.Time
}

func (e *PostingEscrow) Hold(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) error {
	escrowed, found, err := e.escrowed.Get(tx, identityOf(posting))
	if err != nil {
		return fmt.Errorf("read escrowed posting: %w", err)
	}
	if found && escrowed.Posting.LastModified > posting.LastModified {
		return nil
	}
	wasReplaced, err := e.escrowed.Put(tx, identityOf(posting), escrowedPosting{
		HeldAt:  e.now(),
		Origin:  origin,
		Posting: posting,
	})
	if err != nil {
		return fmt.Errorf("escrow posting: %w", err)
	}
	if !wasReplaced {
		e.observer.ObserveHeld(tx, 1)
	}

	return nil
}

func identityOf(posting yacymodel.RWIPosting) postingIdentity {
	return postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
}

func (e *PostingEscrow) PostingsWaitingFor(
	tx *vault.Txn,
	url yacymodel.URLHash,
) ([]WaitingPosting, error) {
	var waiting []WaitingPosting
	err := e.escrowed.Scan(
		tx,
		everyPostingWaitingFor(url),
		func(identity postingIdentity, escrowed escrowedPosting) (bool, error) {
			escrowed.Posting.WordHash = identity.Word
			escrowed.Posting.URLHash = identity.URL
			waiting = append(
				waiting,
				WaitingPosting{Origin: escrowed.Origin, Posting: escrowed.Posting},
			)

			return true, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read postings waiting for url: %w", err)
	}

	return waiting, nil
}

func (e *PostingEscrow) Release(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := e.drop(tx, identityOf(posting)); err != nil {
		return err
	}
	e.observer.ObserveReleased(tx, 1)

	return nil
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

func (e *PostingEscrow) AtCapacity(ctx context.Context) (bool, error) {
	var count int
	err := e.vault.View(ctx, func(tx *vault.Txn) error {
		held, err := e.Count(tx)
		count = held

		return err
	})
	if err != nil {
		return false, err
	}

	return count >= e.capacity, nil
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
