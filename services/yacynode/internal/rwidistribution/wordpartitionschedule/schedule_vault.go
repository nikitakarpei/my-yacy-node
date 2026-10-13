package wordpartitionschedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	wordPartitionEntryBucket vault.Name = "rwidistribution_word_partition_holders"
	scheduleBucket           vault.Name = "rwidistribution_word_partition_schedule"
)

const scheduleVersion = "1"

func registerWordPartitionEntries(
	ctx context.Context,
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (
	*vault.Collection[yacymodel.WordPartition, wordPartitionEntry],
	*vault.ValueIndex[ScheduledWordPartition, ScheduledWordPartition],
	error,
) {
	schedule, err := v.RegisterValueIndex(
		scheduleBucket,
		scheduleVersion,
		scheduleKeyLayout,
		scheduledWordPartitionItself,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register word partition schedule: %w", err)
	}
	entries, err := v.RegisterProjectedCollection(
		ctx,
		wordPartitionEntryBucket,
		wordPartitionKeyLayout,
		wordPartitionEntryValueCodec{},
		vault.ProjectedAs(scheduledWordPartitionFor(partitions), schedule),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register word partition holders: %w", err)
	}

	return entries, schedule, nil
}

func scheduledWordPartitionFor(
	partitions yacymodel.DHTRingPartitions,
) func(yacymodel.WordPartition, wordPartitionEntry) ScheduledWordPartition {
	return func(
		wordPartition yacymodel.WordPartition,
		entry wordPartitionEntry,
	) ScheduledWordPartition {
		return ScheduledWordPartition{
			WordPartition: wordPartition,
			slice: scheduleSliceOf(
				yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions),
			),
			dueAt: entry.dueAt,
		}
	}
}

func scheduledWordPartitionItself(scheduled ScheduledWordPartition) ScheduledWordPartition {
	return scheduled
}

var wordPartitionKeyLayout = vault.PairKey(
	hashkeypart.Hash,
	vault.IntegerKeyPart,
).KeyLayoutFor(
	func(wordPartition yacymodel.WordPartition) (yacymodel.Hash, int64) {
		return wordPartition.Word, partitionKeyOf(wordPartition.Partition)
	},
	func(word yacymodel.Hash, partition int64) yacymodel.WordPartition {
		return yacymodel.WordPartition{Word: word, Partition: partitionFromKey(partition)}
	},
)

var scheduleKeyParts = vault.QuadKey(
	vault.IntegerKeyPart,
	vault.TimeKeyPart,
	hashkeypart.Hash,
	vault.IntegerKeyPart,
)

var scheduleKeyLayout = scheduleKeyParts.KeyLayoutFor(
	func(scheduled ScheduledWordPartition) (int64, time.Time, yacymodel.Hash, int64) {
		return sliceKeyOf(scheduled.slice),
			scheduled.dueAt,
			scheduled.WordPartition.Word,
			partitionKeyOf(scheduled.WordPartition.Partition)
	},
	func(slice int64, dueAt time.Time, word yacymodel.Hash, partition int64) ScheduledWordPartition {
		return ScheduledWordPartition{
			slice: scheduleSlice(slice),
			dueAt: dueAt,
			WordPartition: yacymodel.WordPartition{
				Word:      word,
				Partition: partitionFromKey(partition),
			},
		}
	},
)

func partitionKeyOf(partition uint) int64 {
	return int64(partition) //nolint:gosec // G115: a partition is below DHTRingPartitions.
}

func partitionFromKey(partitionKey int64) uint {
	return uint(partitionKey) //nolint:gosec // G115: keys hold partitions only.
}

func sliceKeyOf(slice scheduleSlice) int64 {
	return int64(slice) //nolint:gosec // G115: a slice is below scheduleSliceCount.
}

func everyWordPartitionInSlice(slice scheduleSlice) vault.KeyRange {
	return scheduleKeyParts.KeysWithFirst(sliceKeyOf(slice))
}

func dueWordPartitionsInSlice(slice scheduleSlice, now time.Time) vault.KeyRange {
	return scheduleKeyParts.KeysWithFirstThroughSecond(sliceKeyOf(slice), now)
}

var errBadWordPartitionEntry = errors.New("bad word partition holders")

type wordPartitionEntryValueCodec struct{}

func (wordPartitionEntryValueCodec) Encode(entry wordPartitionEntry) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(len(entry.holders))
	for _, holder := range entry.holders {
		stored.Fixed(holder.Bytes())
	}
	stored.Time(entry.dueAt)
	stored.Varint(int64(entry.retryDelay))

	return stored.Record(), nil
}

func (wordPartitionEntryValueCodec) Decode(raw []byte) (wordPartitionEntry, error) {
	stored := storedfields.ReaderOf(raw, errBadWordPartitionEntry)
	holders := holdersFrom(stored)
	dueAt := stored.Time("due at")
	retryDelay := time.Duration(stored.Varint("retry delay"))
	if err := stored.Err(); err != nil {
		return wordPartitionEntry{}, err
	}

	return wordPartitionEntry{holders: holders, dueAt: dueAt, retryDelay: retryDelay}, nil
}

func holdersFrom(stored *storedfields.Reader) []yacymodel.Hash {
	count := stored.Count("holder count")
	holdersThatFit := stored.BytesLeft() / yacymodel.HashByteLength
	holders := make([]yacymodel.Hash, 0, min(count, holdersThatFit))
	for range count {
		holder, err := yacymodel.ParseHashBytes(stored.Fixed("holder", yacymodel.HashByteLength))
		if err != nil {
			stored.Reject("holder", err)

			return nil
		}
		holders = append(holders, holder)
	}

	return holders
}
