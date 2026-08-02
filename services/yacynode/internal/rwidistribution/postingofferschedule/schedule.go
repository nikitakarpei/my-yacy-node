// Package postingofferschedule tracks when each stored posting is next due for
// a distribution offer, ordered so the earliest-due posting can be found
// without scanning every posting. A posting that misses its redundancy comes
// back after the offer interval this package holds for it, and that interval
// widens on every further miss.
package postingofferschedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	orderBucket         vault.Name = "rwidistribution_offer_order"
	dueBucket           vault.Name = "rwidistribution_offer_due"
	offerIntervalBucket vault.Name = "rwidistribution_offer_interval"
)

type Observer interface {
	ObserveScheduledPostings(postings int)
	ObserveLongestOfferLateness(lateness time.Duration)
}

type Schedule struct {
	vault          *vault.Vault
	order          *vault.Collection[struct{}]
	dueTimes       *vault.Collection[time.Time]
	offerIntervals *vault.Collection[time.Duration]
	now            func() time.Time
	observer       Observer
}

func Open(v *vault.Vault, now func() time.Time, observer Observer) (*Schedule, error) {
	order, err := vault.Register(v, orderBucket, presenceCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer order: %w", err)
	}
	dueTimes, err := vault.Register(v, dueBucket, dueAtCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer due: %w", err)
	}
	offerIntervals, err := vault.Register(v, offerIntervalBucket, offerIntervalCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer interval: %w", err)
	}

	return &Schedule{
		vault:          v,
		order:          order,
		dueTimes:       dueTimes,
		offerIntervals: offerIntervals,
		now:            now,
		observer:       observer,
	}, nil
}

func (s *Schedule) PostingStored(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	posting := postingidentity.IdentityOf(word, url)
	if err := s.forgetDueAt(tx, posting); err != nil {
		return err
	}

	return s.setDueAt(tx, posting, s.now())
}

func (s *Schedule) forgetDueAt(tx *vault.Txn, posting postingidentity.Identity) error {
	dueAt, found, err := s.dueAt(tx, posting)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return s.clearDueAt(tx, posting, dueAt)
}

func (s *Schedule) dueAt(
	tx *vault.Txn,
	posting postingidentity.Identity,
) (time.Time, bool, error) {
	dueAt, found, err := s.dueTimes.Get(tx, posting.Key())
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read offer due: %w", err)
	}

	return dueAt, found, nil
}

func (s *Schedule) clearDueAt(
	tx *vault.Txn,
	posting postingidentity.Identity,
	dueAt time.Time,
) error {
	if _, err := s.order.Delete(tx, orderKeyFor(posting, dueAt)); err != nil {
		return fmt.Errorf("drop offer order: %w", err)
	}
	if _, err := s.dueTimes.Delete(tx, posting.Key()); err != nil {
		return fmt.Errorf("drop offer due: %w", err)
	}

	return nil
}

func (s *Schedule) setDueAt(
	tx *vault.Txn,
	posting postingidentity.Identity,
	dueAt time.Time,
) error {
	if err := s.order.Put(tx, orderKeyFor(posting, dueAt), struct{}{}); err != nil {
		return fmt.Errorf("record offer order: %w", err)
	}
	if err := s.dueTimes.Put(tx, posting.Key(), dueAt); err != nil {
		return fmt.Errorf("record offer due: %w", err)
	}

	return nil
}

func (s *Schedule) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	posting := postingidentity.IdentityOf(word, url)
	if err := s.forgetDueAt(tx, posting); err != nil {
		return err
	}

	return s.forgetOfferInterval(tx, posting)
}

func (s *Schedule) forgetOfferInterval(tx *vault.Txn, posting postingidentity.Identity) error {
	if _, err := s.offerIntervals.Delete(tx, posting.Key()); err != nil {
		return fmt.Errorf("drop offer retry wait: %w", err)
	}

	return nil
}

func (s *Schedule) SetNextOfferAfterRedundancyMet(
	tx *vault.Txn,
	posting postingidentity.Identity,
	interval OfferInterval,
) error {
	if err := s.forgetOfferInterval(tx, posting); err != nil {
		return err
	}

	return s.reschedule(tx, posting, s.now().Add(interval.Longest))
}

func (s *Schedule) reschedule(
	tx *vault.Txn,
	posting postingidentity.Identity,
	dueAt time.Time,
) error {
	previousDueAt, found, err := s.dueAt(tx, posting)
	if err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}
	if !found {
		return nil
	}
	if err := s.clearDueAt(tx, posting, previousDueAt); err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}

	return s.setDueAt(tx, posting, dueAt)
}

func (s *Schedule) SetNextOfferAfterRedundancyMissed(
	tx *vault.Txn,
	posting postingidentity.Identity,
	interval OfferInterval,
	requestedPause time.Duration,
) error {
	widenedInterval, err := s.widenedOfferInterval(tx, posting, interval)
	if err != nil {
		return err
	}

	return s.reschedule(tx, posting, s.now().Add(max(widenedInterval, requestedPause)))
}

func (s *Schedule) widenedOfferInterval(
	tx *vault.Txn,
	posting postingidentity.Identity,
	interval OfferInterval,
) (time.Duration, error) {
	postingScheduled, err := s.IsScheduled(tx, posting)
	if err != nil {
		return 0, fmt.Errorf("read offer schedule: %w", err)
	}
	if !postingScheduled {
		return 0, nil
	}

	key := posting.Key()
	previousInterval, _, err := s.offerIntervals.Get(tx, key)
	if err != nil {
		return 0, fmt.Errorf("read offer interval: %w", err)
	}

	widenedInterval := interval.widenedFrom(previousInterval)
	if err := s.offerIntervals.Put(tx, key, widenedInterval); err != nil {
		return 0, fmt.Errorf("record offer interval: %w", err)
	}

	return widenedInterval, nil
}

func (s *Schedule) IsScheduled(
	tx *vault.Txn,
	posting postingidentity.Identity,
) (bool, error) {
	_, found, err := s.dueAt(tx, posting)

	return found, err
}

func (s *Schedule) DuePostings(
	ctx context.Context,
	limit int,
) ([]postingidentity.Identity, error) {
	if limit <= 0 {
		return nil, nil
	}

	now := s.now()
	duePostings := make([]postingidentity.Identity, 0, limit)
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		return s.order.Scan(tx, nil, func(key vault.Key, _ struct{}) (bool, error) {
			scheduledOffer, err := parseOrderKey(key)
			if err != nil {
				return false, err
			}
			if scheduledOffer.At.After(now) {
				return false, nil
			}
			duePostings = append(duePostings, scheduledOffer.Posting)

			return len(duePostings) < limit, nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("select due postings: %w", err)
	}

	return duePostings, nil
}

func (s *Schedule) ObserveBacklog(ctx context.Context) {
	s.observeScheduledPostings(ctx)
	s.observeLongestOfferLateness(ctx)
}

func (s *Schedule) observeScheduledPostings(ctx context.Context) {
	var scheduledPostings int
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		scheduledPostings, err = s.order.Len(tx)

		return err
	})
	if err != nil {
		slog.WarnContext(ctx, "scheduled postings not read", slog.Any("error", err))

		return
	}

	s.observer.ObserveScheduledPostings(scheduledPostings)
}

func (s *Schedule) observeLongestOfferLateness(ctx context.Context) {
	earliestDueAt, found, err := s.earliestOfferDueAt(ctx)
	if err != nil {
		slog.WarnContext(ctx, "earliest offer due time not read", slog.Any("error", err))

		return
	}
	if !found {
		s.observer.ObserveLongestOfferLateness(0)

		return
	}

	s.observer.ObserveLongestOfferLateness(max(s.now().Sub(earliestDueAt), 0))
}

func (s *Schedule) earliestOfferDueAt(ctx context.Context) (time.Time, bool, error) {
	var (
		earliestDueAt time.Time
		found         bool
	)
	err := s.vault.View(ctx, func(tx *vault.Txn) error {
		return s.order.Scan(tx, nil, func(key vault.Key, _ struct{}) (bool, error) {
			scheduledOffer, err := parseOrderKey(key)
			if err != nil {
				return false, err
			}
			earliestDueAt = scheduledOffer.At
			found = true

			return false, nil
		})
	})
	if err != nil {
		return time.Time{}, false, fmt.Errorf("select earliest offer due: %w", err)
	}

	return earliestDueAt, found, nil
}

var _ rwipostings.PostingObserver = (*Schedule)(nil)
