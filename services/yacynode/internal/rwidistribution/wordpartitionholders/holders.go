// Package wordpartitionholders owns the record of the peers that hold a copy of
// the postings of each word partition, and when each word partition is next due
// for a refresh. A word partition short of redundancy is in retry: it is due
// again after a retry delay that doubles on each try, up to the refresh
// interval. Due refreshes are read ring sector by ring sector, oldest first in
// each, after the last one read.
package wordpartitionholders

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartitionRecord struct {
	holders    []yacymodel.Hash
	refreshAt  time.Time
	retryDelay time.Duration
}

type ScheduledRefresh struct {
	WordPartition yacymodel.WordPartition
	sector        yacymodel.DHTRingSector
	refreshAt     time.Time
}

type Holders struct {
	records         *vault.Collection[yacymodel.WordPartition, wordPartitionRecord]
	refreshOrder    *vault.Set[ScheduledRefresh]
	partitions      yacymodel.DHTRingPartitions
	refreshInterval time.Duration
	firstRetryDelay time.Duration
	now             func() time.Time
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	refreshInterval time.Duration,
	firstRetryDelay time.Duration,
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
		firstRetryDelay: firstRetryDelay,
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

func (h *Holders) IsInRetry(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (bool, error) {
	record, _, err := h.records.Get(tx, wordPartition)
	if err != nil {
		return false, fmt.Errorf("read word partition retry delay: %w", err)
	}

	return record.retryDelay > 0, nil
}

func (h *Holders) RecordAtRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	return h.record(tx, wordPartition, wordPartitionRecord{
		holders:   holders,
		refreshAt: h.now().Add(h.refreshInterval),
	})
}

func (h *Holders) RecordShortOfRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	retryDelay, err := h.retryDelayFor(tx, wordPartition)
	if err != nil {
		return err
	}

	return h.record(tx, wordPartition, wordPartitionRecord{
		holders:    holders,
		refreshAt:  h.now().Add(retryDelay),
		retryDelay: retryDelay,
	})
}

func (h *Holders) retryDelayFor(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (time.Duration, error) {
	previousRecord, _, err := h.records.Get(tx, wordPartition)
	if err != nil {
		return 0, fmt.Errorf("read word partition retry delay: %w", err)
	}

	return min(max(previousRecord.retryDelay*2, h.firstRetryDelay), h.refreshInterval), nil
}

func (h *Holders) record(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	record wordPartitionRecord,
) error {
	previousRefreshAt, found, err := h.replaceRecord(tx, wordPartition, record)
	if err != nil {
		return err
	}
	if found {
		if err := h.unscheduleRefresh(tx, wordPartition, previousRefreshAt); err != nil {
			return err
		}
	}

	return h.scheduleRefresh(tx, wordPartition, record.refreshAt)
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
) ScheduledRefresh {
	return ScheduledRefresh{
		sector: yacymodel.DHTRingSectorOf(
			yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.partitions),
		),
		refreshAt:     refreshAt,
		WordPartition: wordPartition,
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

func (h *Holders) DueRefreshesAfter(
	tx *vault.Txn,
	lastReadRefresh ScheduledRefresh,
	refreshesBudget int,
) (dueRefreshes []ScheduledRefresh, reachedEnd bool, err error) {
	now := h.now()
	stoppedOnBudget := false
	for sector := lastReadRefresh.sector; sector < yacymodel.DHTRingSectorCount &&
		!stoppedOnBudget; sector++ {
		if err := h.refreshOrder.Scan(
			tx,
			refreshesInSectorDueBy(
				sector,
				now,
			).After(refreshOrderKeyLayout.Encode(lastReadRefresh)),
			func(refresh ScheduledRefresh) (bool, error) {
				if len(dueRefreshes) == refreshesBudget {
					stoppedOnBudget = true

					return false, nil
				}
				dueRefreshes = append(dueRefreshes, refresh)

				return true, nil
			},
		); err != nil {
			return nil, false, fmt.Errorf("select due word partition refreshes: %w", err)
		}
	}

	return dueRefreshes, !stoppedOnBudget, nil
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
			func(refresh ScheduledRefresh) (bool, error) {
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
