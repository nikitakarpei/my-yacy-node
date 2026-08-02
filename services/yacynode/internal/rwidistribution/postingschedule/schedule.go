// Package postingschedule tracks when each stored posting is next due for a
// distribution offer, ordered so the earliest-due posting can be found without
// scanning every posting.
package postingschedule

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	orderBucket vault.Name = "rwidistribution_offer_order"
	dueBucket   vault.Name = "rwidistribution_offer_due"
)

type Identity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type Schedule struct {
	vault *vault.Vault
	order *vault.Collection[struct{}]
	due   *vault.Collection[time.Time]
	now   func() time.Time
}

func Open(v *vault.Vault, now func() time.Time) (*Schedule, error) {
	order, err := vault.Register(v, orderBucket, presenceCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer order: %w", err)
	}
	due, err := vault.Register(v, dueBucket, dueAtCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer due: %w", err)
	}

	return &Schedule{vault: v, order: order, due: due, now: now}, nil
}

func (s *Schedule) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if err := s.forget(tx, word, url); err != nil {
		return err
	}

	return s.setDueAt(tx, word, url, s.now())
}

func (s *Schedule) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return s.forget(tx, word, url)
}

func (s *Schedule) forget(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	at, found, err := s.dueAt(tx, word, url)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return s.clearDueAt(tx, word, url, at)
}

// Reschedule re-arms an existing due entry. It is a no-op when the posting
// has no due row, so a schedule row purged mid-cycle is not resurrected.
func (s *Schedule) Reschedule(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	err := s.vault.Update(ctx, func(tx *vault.Txn) error {
		previous, found, err := s.dueAt(tx, word, url)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		if err := s.clearDueAt(tx, word, url, previous); err != nil {
			return err
		}

		return s.setDueAt(tx, word, url, at)
	})
	if err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}

	return nil
}

// Scheduled reports whether a posting currently has a due entry.
func (s *Schedule) Scheduled(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	_, found, err := s.dueAt(tx, word, url)

	return found, err
}

func (s *Schedule) dueAt(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (time.Time, bool, error) {
	at, found, err := s.due.Get(tx, PostingKey(word, url))
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read offer due: %w", err)
	}

	return at, found, nil
}

func (s *Schedule) setDueAt(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	if err := s.order.Put(tx, orderKey(at, word, url), struct{}{}); err != nil {
		return fmt.Errorf("record offer order: %w", err)
	}
	if err := s.due.Put(tx, PostingKey(word, url), at); err != nil {
		return fmt.Errorf("record offer due: %w", err)
	}

	return nil
}

func (s *Schedule) clearDueAt(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	if _, err := s.order.Delete(tx, orderKey(at, word, url)); err != nil {
		return fmt.Errorf("drop offer order: %w", err)
	}
	if _, err := s.due.Delete(tx, PostingKey(word, url)); err != nil {
		return fmt.Errorf("drop offer due: %w", err)
	}

	return nil
}

func (s *Schedule) DuePostings(
	ctx context.Context,
	limit int,
) ([]Identity, error) {
	if limit <= 0 {
		return nil, nil
	}

	now := s.now()
	due := make([]Identity, 0, limit)
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		return s.order.Scan(tx, nil, func(key vault.Key, _ struct{}) (bool, error) {
			scheduled, err := parseOrderKey(key)
			if err != nil {
				return false, err
			}
			if scheduled.At.After(now) {
				return false, nil
			}
			due = append(due, scheduled.Posting)

			return len(due) < limit, nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("select due postings: %w", err)
	}

	return due, nil
}

// OldestDueAt returns the due time of the earliest-scheduled posting still
// awaiting an offer, or false if the schedule is empty.
func (s *Schedule) OldestDueAt(ctx context.Context) (time.Time, bool, error) {
	var (
		oldest time.Time
		found  bool
	)
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		return s.order.Scan(tx, nil, func(key vault.Key, _ struct{}) (bool, error) {
			scheduled, err := parseOrderKey(key)
			if err != nil {
				return false, err
			}
			oldest = scheduled.At
			found = true

			return false, nil
		})
	})
	if err != nil {
		return time.Time{}, false, fmt.Errorf("select oldest due posting: %w", err)
	}

	return oldest, found, nil
}

var _ rwipostings.PostingObserver = (*Schedule)(nil)
