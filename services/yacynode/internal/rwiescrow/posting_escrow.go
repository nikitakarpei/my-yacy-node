package rwiescrow

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type escrowedPosting struct {
	HeldAt  time.Time
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
	admitter rwipostings.PostingAdmitter
	observer HoldObserver
	capacity int
	now      func() time.Time
}

func (e *PostingEscrow) Hold(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	wasReplaced, err := e.escrowed.Put(tx, identityOf(posting), escrowedPosting{
		HeldAt:  e.now(),
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

func (e *PostingEscrow) URLStored(tx *vault.Txn, hash yacymodel.URLHash) error {
	waiting, err := e.postingsWaitingFor(tx, hash)
	if err != nil {
		return err
	}
	if len(waiting) == 0 {
		return nil
	}

	for _, escrowed := range waiting {
		if err := e.release(tx, escrowed); err != nil {
			return err
		}
	}
	e.observer.ObserveReleased(tx, len(waiting))

	return nil
}

func (e *PostingEscrow) URLPurged(*vault.Txn, yacymodel.URLHash) error {
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

func (e *PostingEscrow) postingsWaitingFor(
	tx *vault.Txn,
	hash yacymodel.URLHash,
) ([]escrowedPosting, error) {
	var waiting []escrowedPosting
	err := e.escrowed.Scan(
		tx,
		everyPostingWaitingFor(hash),
		func(identity postingIdentity, escrowed escrowedPosting) (bool, error) {
			escrowed.Posting.WordHash = identity.Word
			escrowed.Posting.URLHash = identity.URL
			waiting = append(waiting, escrowed)

			return true, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read postings waiting for url: %w", err)
	}

	return waiting, nil
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

func (e *PostingEscrow) release(tx *vault.Txn, escrowed escrowedPosting) error {
	if err := e.admitter.Admit(tx, escrowed.Posting); err != nil {
		return fmt.Errorf("admit released posting: %w", err)
	}

	return e.drop(tx, identityOf(escrowed.Posting))
}

func (e *PostingEscrow) drop(tx *vault.Txn, identity postingIdentity) error {
	if _, err := e.escrowed.Delete(tx, identity); err != nil {
		return fmt.Errorf("drop escrowed posting: %w", err)
	}

	return nil
}
