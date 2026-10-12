// Package wordpartitionholders owns the record of the peers that hold a copy of
// the postings of each word partition, and when each word partition is next due
// for a refresh. The due refreshes are read one ring sector at a time.
package wordpartitionholders

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartitionRecord struct {
	holders   []yacymodel.Hash
	refreshAt time.Time
}

type scheduledRefresh struct {
	sector        yacymodel.DHTRingSector
	refreshAt     time.Time
	wordPartition yacymodel.WordPartition
}

type Holders struct {
	records         *vault.Collection[yacymodel.WordPartition, wordPartitionRecord]
	refreshOrder    *vault.Set[scheduledRefresh]
	partitions      yacymodel.DHTRingPartitions
	refreshInterval time.Duration
	retryDelay      time.Duration
	now             func() time.Time
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	refreshInterval time.Duration,
	retryDelay time.Duration,
	now func() time.Time,
) (*Holders, error) {
	records, err := registerWordPartitionRecords(v)
	if err != nil {
		return nil, err
	}
	refreshOrder, err := registerRefreshOrder(v)
	if err != nil {
		return nil, err
	}

	return &Holders{
		records:         records,
		refreshOrder:    refreshOrder,
		partitions:      partitions,
		refreshInterval: refreshInterval,
		retryDelay:      retryDelay,
		now:             now,
	}, nil
}

func (h *Holders) Of(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) ([]yacymodel.Hash, error) {
	record, _, err := h.records.Get(tx, wordPartition)
	if err != nil {
		return nil, fmt.Errorf("read word partition holders: %w", err)
	}

	return record.holders, nil
}

func (h *Holders) RecordAtRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	return h.record(tx, wordPartition, holders, h.now().Add(h.refreshInterval))
}

func (h *Holders) RecordShortOfRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	return h.record(tx, wordPartition, holders, h.now().Add(h.retryDelay))
}

func (h *Holders) record(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
	refreshAt time.Time,
) error {
	previousRefreshAt, found, err := h.replaceRecord(
		tx,
		wordPartition,
		wordPartitionRecord{holders: holders, refreshAt: refreshAt},
	)
	if err != nil {
		return err
	}
	if found {
		if err := h.unscheduleRefresh(tx, wordPartition, previousRefreshAt); err != nil {
			return err
		}
	}

	return h.scheduleRefresh(tx, wordPartition, refreshAt)
}

func (h *Holders) replaceRecord(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	record wordPartitionRecord,
) (previousRefreshAt time.Time, wasReplaced bool, err error) {
	previousRecord, wasReplaced, err := h.records.PutReturning(tx, wordPartition, record)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("record word partition holders: %w", err)
	}

	return previousRecord.refreshAt, wasReplaced, nil
}

func (h *Holders) unscheduleRefresh(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	refreshAt time.Time,
) error {
	if _, err := h.refreshOrder.Remove(
		tx,
		h.scheduledRefreshOf(wordPartition, refreshAt),
	); err != nil {
		return fmt.Errorf("drop word partition refresh: %w", err)
	}

	return nil
}

func (h *Holders) scheduledRefreshOf(
	wordPartition yacymodel.WordPartition,
	refreshAt time.Time,
) scheduledRefresh {
	return scheduledRefresh{
		sector: yacymodel.DHTRingSectorOf(
			yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.partitions),
		),
		refreshAt:     refreshAt,
		wordPartition: wordPartition,
	}
}

func (h *Holders) scheduleRefresh(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	refreshAt time.Time,
) error {
	if _, err := h.refreshOrder.Add(
		tx,
		h.scheduledRefreshOf(wordPartition, refreshAt),
	); err != nil {
		return fmt.Errorf("record word partition refresh: %w", err)
	}

	return nil
}

func (h *Holders) Forget(tx *vault.Txn, wordPartition yacymodel.WordPartition) error {
	previousRefreshAt, found, err := h.deleteRecord(tx, wordPartition)
	if err != nil || !found {
		return err
	}

	return h.unscheduleRefresh(tx, wordPartition, previousRefreshAt)
}

func (h *Holders) deleteRecord(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (previousRefreshAt time.Time, wasDeleted bool, err error) {
	previousRecord, wasDeleted, err := h.records.DeleteReturning(tx, wordPartition)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("drop word partition holders: %w", err)
	}

	return previousRecord.refreshAt, wasDeleted, nil
}

func (h *Holders) DueRefreshesIn(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
	limit int,
) ([]yacymodel.WordPartition, error) {
	dueWordPartitions := make([]yacymodel.WordPartition, 0, max(limit, 0))
	if err := h.refreshOrder.Scan(
		tx,
		refreshesInSectorDueBy(sector, h.now()),
		func(refresh scheduledRefresh) (bool, error) {
			dueWordPartitions = append(dueWordPartitions, refresh.wordPartition)

			return len(dueWordPartitions) < limit, nil
		},
	); err != nil {
		return nil, fmt.Errorf("select due word partition refreshes: %w", err)
	}

	return dueWordPartitions, nil
}

func (h *Holders) EarliestRefreshAt(tx *vault.Txn) (time.Time, bool, error) {
	var (
		earliestRefreshAt time.Time
		found             bool
	)
	for sector := range yacymodel.DHTRingSectorCount {
		if err := h.refreshOrder.Scan(
			tx,
			everyRefreshInSector(sector),
			func(refresh scheduledRefresh) (bool, error) {
				if !found || refresh.refreshAt.Before(earliestRefreshAt) {
					earliestRefreshAt, found = refresh.refreshAt, true
				}

				return false, nil
			},
		); err != nil {
			return time.Time{}, false, fmt.Errorf("read earliest word partition refresh: %w", err)
		}
	}

	return earliestRefreshAt, found, nil
}
