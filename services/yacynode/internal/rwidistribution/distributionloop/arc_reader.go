package distributionloop

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
)

type readEntry struct {
	scheduled wordpartitionschedule.ScheduledWordPartition
	spanIndex int
}

type arcRead struct {
	entries    []readEntry
	reachedEnd bool
}

type arcReader struct {
	vault                  *vault.Vault
	schedule               *wordpartitionschedule.Schedule
	amountOfWordPartitions int
}

func (r arcReader) dueEntriesFrom(
	ctx context.Context,
	spans []yacymodel.DHTRingSpan,
	position arcPosition,
) (arcRead, error) {
	var read arcRead
	err := r.vault.View(ctx, func(tx *vault.Txn) error {
		read = arcRead{}
		for spanIndex := position.spanIndex; spanIndex < len(spans); spanIndex++ {
			if len(read.entries) == r.amountOfWordPartitions {
				return nil
			}
			reachedSpanEnd, err := r.addDueEntriesWithin(tx, spans, spanIndex, position, &read)
			if err != nil || !reachedSpanEnd {
				return err
			}
		}
		read.reachedEnd = true

		return nil
	})

	return read, err
}

func (r arcReader) addDueEntriesWithin(
	tx *vault.Txn,
	spans []yacymodel.DHTRingSpan,
	spanIndex int,
	position arcPosition,
	read *arcRead,
) (reachedSpanEnd bool, err error) {
	var cursor wordpartitionschedule.ScheduledWordPartition
	if spanIndex == position.spanIndex {
		cursor = position.cursor
	}
	due, reachedSpanEnd, err := r.schedule.WordPartitionsDueWithin(
		tx, spans[spanIndex], cursor, r.amountOfWordPartitions-len(read.entries))
	if err != nil {
		return false, err
	}
	for _, scheduled := range due {
		read.entries = append(read.entries, readEntry{scheduled: scheduled, spanIndex: spanIndex})
	}

	return reachedSpanEnd, nil
}

func (r arcReader) earliestDueWithin(
	ctx context.Context,
	spans []yacymodel.DHTRingSpan,
) (earliestDueAt time.Time, isAnyDue bool, err error) {
	err = r.vault.View(ctx, func(tx *vault.Txn) error {
		for _, span := range spans {
			dueAt, found, err := r.schedule.EarliestDueWithin(tx, span)
			if err != nil {
				return err
			}
			if found && (!isAnyDue || dueAt.Before(earliestDueAt)) {
				earliestDueAt, isAnyDue = dueAt, true
			}
		}

		return nil
	})

	return earliestDueAt, isAnyDue, err
}
