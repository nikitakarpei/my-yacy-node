package rwidistribution

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	postingOfferOrderBucket vault.Name = "rwidistribution_offer_order"
	postingOfferDueBucket   vault.Name = "rwidistribution_offer_due"
)

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type postingOfferSchedule struct {
	vault *vault.Vault
	order *vault.Collection[struct{}]
	due   *vault.Collection[time.Time]
	now   func() time.Time
}

func openPostingOfferSchedule(v *vault.Vault, now func() time.Time) (*postingOfferSchedule, error) {
	order, err := vault.Register(v, postingOfferOrderBucket, presenceCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer order: %w", err)
	}
	due, err := vault.Register(v, postingOfferDueBucket, dueAtCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer due: %w", err)
	}

	return &postingOfferSchedule{vault: v, order: order, due: due, now: now}, nil
}

func (s *postingOfferSchedule) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return s.reschedule(tx, word, url, s.now())
}

func (s *postingOfferSchedule) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return s.forget(tx, word, url)
}

func (s *postingOfferSchedule) Forget(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	err := s.vault.Update(ctx, func(tx *vault.Txn) error {
		return s.forget(tx, word, url)
	})
	if err != nil {
		return fmt.Errorf("forget posting: %w", err)
	}

	return nil
}

func (s *postingOfferSchedule) forget(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	at, found, err := s.due.Get(tx, postingKey(word, url))
	if err != nil {
		return fmt.Errorf("read offer due: %w", err)
	}
	if !found {
		return nil
	}

	return s.clear(tx, word, url, at)
}

func (s *postingOfferSchedule) Reschedule(
	ctx context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	err := s.vault.Update(ctx, func(tx *vault.Txn) error {
		return s.reschedule(tx, word, url, at)
	})
	if err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}

	return nil
}

func (s *postingOfferSchedule) reschedule(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	previous, found, err := s.due.Get(tx, postingKey(word, url))
	if err != nil {
		return fmt.Errorf("read offer due: %w", err)
	}
	if found {
		if _, err := s.order.Delete(tx, orderKey(previous, word, url)); err != nil {
			return fmt.Errorf("drop offer order: %w", err)
		}
	}

	if err := s.order.Put(tx, orderKey(at, word, url), struct{}{}); err != nil {
		return fmt.Errorf("record offer order: %w", err)
	}
	if err := s.due.Put(tx, postingKey(word, url), at); err != nil {
		return fmt.Errorf("record offer due: %w", err)
	}

	return nil
}

func (s *postingOfferSchedule) clear(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	at time.Time,
) error {
	if _, err := s.order.Delete(tx, orderKey(at, word, url)); err != nil {
		return fmt.Errorf("drop offer order: %w", err)
	}
	if _, err := s.due.Delete(tx, postingKey(word, url)); err != nil {
		return fmt.Errorf("drop offer due: %w", err)
	}

	return nil
}

func (s *postingOfferSchedule) DuePostings(
	ctx context.Context,
	limit int,
) ([]postingIdentity, error) {
	if limit <= 0 {
		return nil, nil
	}

	now := s.now()
	due := make([]postingIdentity, 0, limit)
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

var _ rwipostings.PostingObserver = (*postingOfferSchedule)(nil)
