// Package postingofferschedule tracks when each stored posting is next due for
// a distribution offer, ordered so the earliest-due posting can be found
// without scanning every posting. A posting that misses its redundancy comes
// back after the offer interval this package stores for it. Postings short of
// their redundancy are due before postings that only refresh their holders.
// Each batch of due postings starts in the sector of the earliest due posting
// and walks the ring from there, so a batch goes to few peers.
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

type offerDue struct {
	Sector yacymodel.DHTRingSector
	At     time.Time
}

type scheduledPostingOffer struct {
	Due      offerDue
	Identity postingidentity.Identity
}

type Schedule struct {
	shortfallOrder *vault.Set[scheduledPostingOffer]
	refreshOrder   *vault.Set[scheduledPostingOffer]
	offerDues      *vault.Collection[postingidentity.Identity, offerDue]
	offerIntervals *vault.Collection[postingidentity.Identity, time.Duration]
	partitions     yacymodel.DHTRingPartitions
	now            func() time.Time
	observer       Observer
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	now func() time.Time,
	observer Observer,
) (*Schedule, error) {
	shortfallOrder, err := registerOfferOrder(v, shortfallOrderBucket)
	if err != nil {
		return nil, err
	}
	refreshOrder, err := registerOfferOrder(v, refreshOrderBucket)
	if err != nil {
		return nil, err
	}
	offerDues, err := registerOfferDues(v)
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
		offerDues:      offerDues,
		offerIntervals: offerIntervals,
		partitions:     partitions,
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
	due, found, err := s.dueOf(tx, identity)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return s.clearDue(tx, identity, due)
}

func (s *Schedule) dueOf(
	tx *vault.Txn,
	identity postingidentity.Identity,
) (offerDue, bool, error) {
	due, found, err := s.offerDues.Get(tx, identity)
	if err != nil {
		return offerDue{}, false, fmt.Errorf("read offer due: %w", err)
	}

	return due, found, nil
}

func (s *Schedule) clearDue(
	tx *vault.Txn,
	identity postingidentity.Identity,
	due offerDue,
) error {
	scheduledOffer := scheduledPostingOffer{Due: due, Identity: identity}
	for _, order := range []*vault.Set[scheduledPostingOffer]{s.shortfallOrder, s.refreshOrder} {
		if _, err := order.Remove(tx, scheduledOffer); err != nil {
			return fmt.Errorf("drop offer order: %w", err)
		}
	}
	if _, err := s.offerDues.Delete(tx, identity); err != nil {
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
	due := offerDue{Sector: s.dhtRingSectorOf(identity), At: dueAt}
	if _, err := order.Add(tx, scheduledPostingOffer{Due: due, Identity: identity}); err != nil {
		return fmt.Errorf("record offer order: %w", err)
	}
	if _, err := s.offerDues.Put(tx, identity, due); err != nil {
		return fmt.Errorf("record offer due: %w", err)
	}

	return nil
}

func (s *Schedule) dhtRingSectorOf(identity postingidentity.Identity) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOfPosting(
		yacymodel.RWIPosting{WordHash: identity.Word, URLHash: identity.URL},
		s.partitions,
	))
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
	previousDue, found, err := s.dueOf(tx, identity)
	if err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}
	if !found {
		return nil
	}
	if err := s.clearDue(tx, identity, previousDue); err != nil {
		return fmt.Errorf("reschedule offer: %w", err)
	}

	return s.setDueAt(tx, order, identity, nextDueAtFrom(previousDue.At))
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
	_, found, err := s.dueOf(tx, identity)

	return found, err
}

func (s *Schedule) DuePostings(
	tx *vault.Txn,
	limit int,
) ([]postingidentity.Identity, error) {
	if limit <= 0 {
		return nil, nil
	}
	firstSector, found, err := s.sectorOfEarliestDuePosting(tx)
	if err != nil || !found {
		return nil, err
	}

	duePostings := make([]postingidentity.Identity, 0, limit)
	duePostings, err = s.appendDuePostingsOf(tx, s.shortfallOrder, firstSector, duePostings, limit)
	if err != nil {
		return nil, err
	}

	return s.appendDuePostingsOf(tx, s.refreshOrder, firstSector, duePostings, limit)
}

func (s *Schedule) sectorOfEarliestDuePosting(
	tx *vault.Txn,
) (yacymodel.DHTRingSector, bool, error) {
	for _, order := range []*vault.Set[scheduledPostingOffer]{s.shortfallOrder, s.refreshOrder} {
		earliestOffer, found, err := earliestOfferOf(tx, order)
		if err != nil {
			return 0, false, err
		}
		if found && !earliestOffer.Due.At.After(s.now()) {
			return earliestOffer.Due.Sector, true, nil
		}
	}

	return 0, false, nil
}

func earliestOfferOf(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
) (scheduledPostingOffer, bool, error) {
	var (
		earliestOffer scheduledPostingOffer
		found         bool
	)
	for sector := range yacymodel.MaxDHTRingSector + 1 {
		if err := order.Scan(
			tx,
			everyOfferIn(sector),
			func(scheduledOffer scheduledPostingOffer) (bool, error) {
				if !found || scheduledOffer.Due.At.Before(earliestOffer.Due.At) {
					earliestOffer, found = scheduledOffer, true
				}

				return false, nil
			},
		); err != nil {
			return scheduledPostingOffer{}, false, fmt.Errorf("select earliest offer due: %w", err)
		}
	}

	return earliestOffer, found, nil
}

func (s *Schedule) appendDuePostingsOf(
	tx *vault.Txn,
	order *vault.Set[scheduledPostingOffer],
	firstSector yacymodel.DHTRingSector,
	duePostings []postingidentity.Identity,
	limit int,
) ([]postingidentity.Identity, error) {
	for _, sector := range dhtRingSectorsFrom(firstSector) {
		if len(duePostings) >= limit {
			return duePostings, nil
		}
		if err := order.Scan(
			tx,
			everyOfferInSectorDueBy(sector, s.now()),
			func(scheduledOffer scheduledPostingOffer) (bool, error) {
				duePostings = append(duePostings, scheduledOffer.Identity)

				return len(duePostings) < limit, nil
			},
		); err != nil {
			return nil, fmt.Errorf("select due postings: %w", err)
		}
	}

	return duePostings, nil
}

func dhtRingSectorsFrom(firstSector yacymodel.DHTRingSector) []yacymodel.DHTRingSector {
	sectorCount := yacymodel.MaxDHTRingSector + 1
	sectors := make([]yacymodel.DHTRingSector, 0, sectorCount)
	for step := range sectorCount {
		sectors = append(sectors, (firstSector+step)%sectorCount)
	}

	return sectors
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
	earliestOffer, found, err := earliestOfferOf(tx, order)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, nil
	}

	return max(s.now().Sub(earliestOffer.Due.At), 0), nil
}

var _ rwipostings.PostingObserver = (*Schedule)(nil)
