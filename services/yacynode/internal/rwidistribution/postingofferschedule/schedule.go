// Package postingofferschedule tracks when each stored posting is next due for
// a distribution offer, ordered so the earliest-due posting can be found
// without scanning every posting. A posting that misses its redundancy comes
// back after the offer interval this package stores for it. Postings short of
// their redundancy are due before postings that only refresh their holders.
package postingofferschedule

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type OfferOrder string

const (
	OfferOrderShortfall OfferOrder = "shortfall"
	OfferOrderRefresh   OfferOrder = "refresh"
)

type Observer interface {
	ObserveScheduledPostings(order string, postings int)
	ObserveLongestOfferLateness(order string, lateness time.Duration)
}

type scheduledPostingOffer struct {
	At       time.Time
	Identity postingidentity.Identity
}

type Schedule struct {
	shortfallOrder *vault.Set[scheduledPostingOffer]
	refreshOrder   *vault.Set[scheduledPostingOffer]
	dueTimes       *vault.Collection[postingidentity.Identity, time.Time]
	offerIntervals *vault.Collection[postingidentity.Identity, time.Duration]
	now            func() time.Time
	observer       Observer
}

func Open(v *vault.Vault, now func() time.Time, observer Observer) (*Schedule, error) {
	shortfallOrder, err := registerOfferOrder(v, shortfallOrderBucket)
	if err != nil {
		return nil, err
	}
	refreshOrder, err := registerOfferOrder(v, refreshOrderBucket)
	if err != nil {
		return nil, err
	}
	dueTimes, err := registerOfferDues(v)
	if err != nil {
		return nil, err
	}
	offerIntervals, err := registerOfferIntervals(v)
	if err != nil {
		return nil, err
	}

	return &Schedule{
		shortfallOrder: shortfallOrder,
		refreshOrder:   refreshOrder,
		dueTimes:       dueTimes,
		offerIntervals: offerIntervals,
		now:            now,
		observer:       observer,
	}, nil
}

func (s *Schedule) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	identity := postingidentity.IdentityOf(posting)
	if err := s.forgetDueAt(tx, identity); err != nil {
		return err
	}

	return s.setDueAt(tx, s.shortfallOrder, identity, s.now())
}

func (s *Schedule) forgetDueAt(tx *vault.Txn, identity postingidentity.Identity) error {
	dueAt, found, err := s.dueAt(tx, identity)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return s.clearDueAt(tx, identity, dueAt)
}

func (s *Schedule) dueAt(
	tx *vault.Txn,
	identity postingidentity.Identity,
) (time.Time, bool, error) {
	dueAt, found, err := s.dueTimes.Get(tx, identity)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("read offer due: %w", err)
	}

	return dueAt, found, nil
}

func (s *Schedule) clearDueAt(
	tx *vault.Txn,
	identity postingidentity.Identity,
	dueAt time.Time,
) error {
	scheduledOffer := scheduledPostingOffer{At: dueAt, Identity: identity}
	for _, order := range []*vault.Set[scheduledPostingOffer]{s.shortfallOrder, s.refreshOrder} {
		if _, err := order.Remove(tx, scheduledOffer); err != nil {
			return fmt.Errorf("drop offer order: %w", err)
		}
	}
	if _, err := s.dueTimes.Delete(tx, identity); err != nil {
		return fmt.Errorf("drop offer due: %w", err)
	}

	return nil
}

func (s *Schedule) setDueAt(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
	identity postingidentity.Identity,
	dueAt time.Time,
) error {
	if _, err := order.Add(tx, scheduledPostingOffer{At: dueAt, Identity: identity}); err != nil {
		return fmt.Errorf("record offer order: %w", err)
	}
	if _, err := s.dueTimes.Put(tx, identity, dueAt); err != nil {
		return fmt.Errorf("record offer due: %w", err)
	}

	return nil
}

func (s *Schedule) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	identity := postingidentity.IdentityOf(posting)
	if err := s.forgetDueAt(tx, identity); err != nil {
		return err
	}

	return s.forgetOfferInterval(tx, identity)
}

func (s *Schedule) forgetOfferInterval(tx *vault.Txn, identity postingidentity.Identity) error {
	if _, err := s.offerIntervals.Delete(tx, identity); err != nil {
		return fmt.Errorf("drop offer interval: %w", err)
	}

	return nil
}

func (s *Schedule) SetNextOfferAfterRedundancyMet(
	tx *vault.Txn,
	identity postingidentity.Identity,
	bounds postingofferinterval.Bounds,
) error {
	if err := s.forgetOfferInterval(tx, identity); err != nil {
		return err
	}

	return s.reschedule(tx, s.refreshOrder, identity, func(previousDueAt time.Time) time.Time {
		return bounds.NextOfferDueFrom(previousDueAt, s.now())
	})
}

func (s *Schedule) reschedule(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
	identity postingidentity.Identity,
	nextDueAtFrom func(previousDueAt time.Time) time.Time,
) error {
	previousDueAt, found, err := s.dueAt(tx, identity)
	if err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}
	if !found {
		return nil
	}
	if err := s.clearDueAt(tx, identity, previousDueAt); err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}

	return s.setDueAt(tx, order, identity, nextDueAtFrom(previousDueAt))
}

func (s *Schedule) SetNextOfferAfterRedundancyMissed(
	tx *vault.Txn,
	identity postingidentity.Identity,
	bounds postingofferinterval.Bounds,
	requestedPause time.Duration,
) error {
	postingScheduled, err := s.IsScheduled(tx, identity)
	if err != nil {
		return fmt.Errorf("read offer schedule: %w", err)
	}
	if !postingScheduled {
		return nil
	}

	previousInterval, _, err := s.offerIntervals.Get(tx, identity)
	if err != nil {
		return fmt.Errorf("read offer interval: %w", err)
	}
	if _, err := s.offerIntervals.Put(
		tx,
		identity,
		bounds.WidenedFrom(previousInterval),
	); err != nil {
		return fmt.Errorf("record offer interval: %w", err)
	}
	pause := bounds.PauseFrom(previousInterval, requestedPause)

	return s.reschedule(tx, s.shortfallOrder, identity, func(time.Time) time.Time {
		return s.now().Add(pause)
	})
}

func (s *Schedule) IsScheduled(
	tx *vault.Txn,
	identity postingidentity.Identity,
) (bool, error) {
	_, found, err := s.dueAt(tx, identity)

	return found, err
}

func (s *Schedule) DuePostings(
	tx *vault.Txn,
	limit int,
) ([]postingidentity.Identity, error) {
	if limit <= 0 {
		return nil, nil
	}

	duePostings := make([]postingidentity.Identity, 0, limit)
	duePostings, err := s.appendDuePostingsOf(tx, s.shortfallOrder, duePostings, limit)
	if err != nil {
		return nil, err
	}

	return s.appendDuePostingsOf(tx, s.refreshOrder, duePostings, limit)
}

func (s *Schedule) appendDuePostingsOf(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
	duePostings []postingidentity.Identity,
	limit int,
) ([]postingidentity.Identity, error) {
	if len(duePostings) >= limit {
		return duePostings, nil
	}
	if err := order.Scan(
		tx,
		everyOfferDueBy(s.now()),
		func(scheduledOffer scheduledPostingOffer) (bool, error) {
			duePostings = append(duePostings, scheduledOffer.Identity)

			return len(duePostings) < limit, nil
		},
	); err != nil {
		return nil, fmt.Errorf("select due postings: %w", err)
	}

	return duePostings, nil
}

func (s *Schedule) ObserveBacklog(tx *vault.Txn) error {
	if err := s.observeBacklogOf(tx, OfferOrderShortfall, s.shortfallOrder); err != nil {
		return err
	}

	return s.observeBacklogOf(tx, OfferOrderRefresh, s.refreshOrder)
}

func (s *Schedule) observeBacklogOf(
	tx *vault.Txn,
	orderName OfferOrder,
	order *vault.Set[scheduledPostingOffer],
) error {
	scheduledPostings, err := order.Len(tx)
	if err != nil {
		return fmt.Errorf("count scheduled postings: %w", err)
	}
	longestLateness, err := s.longestOfferLatenessOf(tx, order)
	if err != nil {
		return err
	}

	s.observer.ObserveScheduledPostings(string(orderName), scheduledPostings)
	s.observer.ObserveLongestOfferLateness(string(orderName), longestLateness)

	return nil
}

func (s *Schedule) longestOfferLatenessOf(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
) (time.Duration, error) {
	earliestDueAt, found, err := earliestOfferDueAtOf(tx, order)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}

	return max(s.now().Sub(earliestDueAt), 0), nil
}

func earliestOfferDueAtOf(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
) (time.Time, bool, error) {
	var (
		earliestDueAt time.Time
		found         bool
	)
	if err := order.Scan(
		tx,
		vault.EveryKey(),
		func(scheduledOffer scheduledPostingOffer) (bool, error) {
			earliestDueAt = scheduledOffer.At
			found = true

			return false, nil
		},
	); err != nil {
		return time.Time{}, false, fmt.Errorf("select earliest offer due: %w", err)
	}

	return earliestDueAt, found, nil
}

var _ rwipostings.PostingObserver = (*Schedule)(nil)
