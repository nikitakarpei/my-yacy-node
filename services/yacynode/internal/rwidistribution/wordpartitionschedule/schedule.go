// Package wordpartitionschedule keeps for each word partition when it is next
// due, when it was last refreshed, and the peers that hold it. Admitting a posting
// makes its word partition due. A send short of redundancy is due again after a
// retry delay that doubles on each try, up to the refresh interval. Due word
// partitions are read by ring span.
package wordpartitionschedule

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var wholeRing = yacymodel.DHTRingSpan{First: 0, Last: yacymodel.MaxDHTRingPosition}

type wordPartitionEntry struct {
	holders     []yacymodel.Hash
	dueAt       time.Time
	retryDelay  time.Duration
	refreshedAt time.Time
}

type ScheduledWordPartition struct {
	WordPartition yacymodel.WordPartition
	slice         scheduleSlice
	dueAt         time.Time
}

func (s ScheduledWordPartition) Sector() yacymodel.DHTRingSector {
	return s.slice.sector()
}

type Schedule struct {
	entries         *vault.Collection[yacymodel.WordPartition, wordPartitionEntry]
	schedule        *vault.ValueIndex[ScheduledWordPartition, ScheduledWordPartition]
	partitions      yacymodel.DHTRingPartitions
	refreshInterval time.Duration
	firstRetryDelay time.Duration
	now             func() time.Time
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	ctx context.Context,
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	refreshInterval time.Duration,
	firstRetryDelay time.Duration,
	now func() time.Time,
) (*Schedule, error) {
	entries, schedule, err := registerWordPartitionEntries(ctx, v, partitions)
	if err != nil {
		return nil, err
	}

	return &Schedule{
		entries:         entries,
		schedule:        schedule,
		partitions:      partitions,
		refreshInterval: refreshInterval,
		firstRetryDelay: firstRetryDelay,
		now:             now,
	}, nil
}

func (s *Schedule) HoldersOf(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) ([]yacymodel.Hash, error) {
	entry, _, err := s.entries.Get(tx, wordPartition)
	if err != nil {
		return nil, fmt.Errorf("read word partition holders: %w", err)
	}

	return entry.holders, nil
}

func (s *Schedule) IsInRetry(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (bool, error) {
	entry, _, err := s.entries.Get(tx, wordPartition)
	if err != nil {
		return false, fmt.Errorf("read word partition retry delay: %w", err)
	}

	return entry.retryDelay > 0, nil
}

func (s *Schedule) IsRefreshDue(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (bool, error) {
	entry, _, err := s.entries.Get(tx, wordPartition)
	if err != nil {
		return false, fmt.Errorf("read word partition refresh time: %w", err)
	}

	return !entry.refreshedAt.Add(s.refreshInterval).After(s.now()), nil
}

func (s *Schedule) RecordRefreshed(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	now := s.now()

	return s.record(tx, wordPartition, wordPartitionEntry{
		holders:     holders,
		dueAt:       now.Add(s.refreshInterval),
		refreshedAt: now,
	})
}

func (s *Schedule) RecordAtRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	previousEntry, err := s.previousEntryOf(tx, wordPartition)
	if err != nil {
		return err
	}

	return s.record(tx, wordPartition, wordPartitionEntry{
		holders:     holders,
		dueAt:       previousEntry.refreshedAt.Add(s.refreshInterval),
		refreshedAt: previousEntry.refreshedAt,
	})
}

func (s *Schedule) RecordShortOfRedundancy(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
) error {
	previousEntry, err := s.previousEntryOf(tx, wordPartition)
	if err != nil {
		return err
	}
	retryDelay := s.retryDelayAfter(previousEntry.retryDelay)

	return s.record(tx, wordPartition, wordPartitionEntry{
		holders:     holders,
		dueAt:       s.now().Add(retryDelay),
		retryDelay:  retryDelay,
		refreshedAt: previousEntry.refreshedAt,
	})
}

func (s *Schedule) previousEntryOf(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (wordPartitionEntry, error) {
	previousEntry, _, err := s.entries.Get(tx, wordPartition)
	if err != nil {
		return wordPartitionEntry{}, fmt.Errorf("read previous word partition holders: %w", err)
	}

	return previousEntry, nil
}

func (s *Schedule) retryDelayAfter(previousRetryDelay time.Duration) time.Duration {
	return min(max(previousRetryDelay*2, s.firstRetryDelay), s.refreshInterval)
}

func (s *Schedule) record(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	entry wordPartitionEntry,
) error {
	if _, err := s.entries.Put(tx, wordPartition, entry); err != nil {
		return fmt.Errorf("record word partition holders: %w", err)
	}

	return nil
}

func (s *Schedule) MakeDue(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (wasMadeDue bool, err error) {
	entry, found, err := s.entries.Get(tx, wordPartition)
	if err != nil {
		return false, fmt.Errorf("read word partition due time: %w", err)
	}
	now := s.now()
	if found && (entry.retryDelay > 0 || !entry.dueAt.After(now)) {
		return false, nil
	}
	entry.dueAt = now

	return true, s.record(tx, wordPartition, entry)
}

func (s *Schedule) Forget(tx *vault.Txn, wordPartition yacymodel.WordPartition) error {
	if _, err := s.entries.Delete(tx, wordPartition); err != nil {
		return fmt.Errorf("drop word partition holders: %w", err)
	}

	return nil
}

func (s *Schedule) WordPartitionsDueWithin(
	tx *vault.Txn,
	span yacymodel.DHTRingSpan,
	lastRead ScheduledWordPartition,
	budget int,
) (due []ScheduledWordPartition, reachedEnd bool, err error) {
	now := s.now()
	stoppedOnBudget := false
	firstSlice := max(scheduleSliceOf(span.First), lastRead.slice)
	for slice := firstSlice; slice <= scheduleSliceOf(span.Last) && !stoppedOnBudget; slice++ {
		if err := s.schedule.Scan(
			tx,
			dueWordPartitionsInSlice(slice, now).After(scheduleKeyLayout.Encode(lastRead)),
			func(scheduled ScheduledWordPartition) (bool, error) {
				if !span.Contains(
					yacymodel.DHTRingPositionOfWordPartition(scheduled.WordPartition, s.partitions),
				) {
					return true, nil
				}
				if len(due) == budget {
					stoppedOnBudget = true

					return false, nil
				}
				due = append(due, scheduled)

				return true, nil
			},
		); err != nil {
			return nil, false, fmt.Errorf("select due word partitions within a span: %w", err)
		}
	}

	return due, !stoppedOnBudget, nil
}

func (s *Schedule) EarliestDueAt(tx *vault.Txn) (time.Time, bool, error) {
	return s.EarliestDueWithin(tx, wholeRing)
}

func (s *Schedule) EarliestDueWithin(
	tx *vault.Txn,
	span yacymodel.DHTRingSpan,
) (earliestDueAt time.Time, found bool, err error) {
	for slice := scheduleSliceOf(span.First); slice <= scheduleSliceOf(span.Last); slice++ {
		if err := s.schedule.Scan(
			tx,
			everyWordPartitionInSlice(slice),
			func(scheduled ScheduledWordPartition) (bool, error) {
				if !span.Contains(
					yacymodel.DHTRingPositionOfWordPartition(scheduled.WordPartition, s.partitions),
				) {
					return true, nil
				}
				if !found || scheduled.dueAt.Before(earliestDueAt) {
					earliestDueAt, found = scheduled.dueAt, true
				}

				return false, nil
			},
		); err != nil {
			return time.Time{}, false, fmt.Errorf("read earliest word partition due time: %w", err)
		}
	}

	return earliestDueAt, found, nil
}
