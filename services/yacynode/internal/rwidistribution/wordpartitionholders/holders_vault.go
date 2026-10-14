package wordpartitionholders

import (
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const (
	wordPartitionRecordBucket vault.Name = "rwidistribution_word_partition_holders"
	refreshOrderBucket        vault.Name = "rwidistribution_word_partition_refresh_order"
)

func registerWordPartitionRecords(
	v *vault.Vault,
) (*vault.Collection[yacymodel.WordPartition, wordPartitionRecord], error) {
	records, err := v.RegisterCollection(
		wordPartitionRecordBucket,
		wordPartitionKeyLayout,
		wordPartitionRecordValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register word partition holders: %w", err)
	}

	return records, nil
}

func registerRefreshOrder(v *vault.Vault) (*vault.Set[ScheduledRefresh], error) {
	refreshOrder, err := v.RegisterSet(refreshOrderBucket, refreshOrderKeyLayout)
	if err != nil {
		return nil, fmt.Errorf("register word partition refresh order: %w", err)
	}

	return refreshOrder, nil
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

var refreshOrderKeyParts = vault.QuadKey(
	vault.IntegerKeyPart,
	vault.TimeKeyPart,
	hashkeypart.Hash,
	vault.IntegerKeyPart,
)

var refreshOrderKeyLayout = refreshOrderKeyParts.KeyLayoutFor(
	func(refresh ScheduledRefresh) (int64, time.Time, yacymodel.Hash, int64) {
		return sectorKeyOf(refresh.sector),
			refresh.refreshAt,
			refresh.WordPartition.Word,
			partitionKeyOf(refresh.WordPartition.Partition)
	},
	func(sector int64, refreshAt time.Time, word yacymodel.Hash, partition int64) ScheduledRefresh {
		return ScheduledRefresh{
			sector:    yacymodel.DHTRingSector(sector),
			refreshAt: refreshAt,
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

func sectorKeyOf(sector yacymodel.DHTRingSector) int64 {
	return int64(sector) //nolint:gosec // G115: a sector is at most MaxDHTRingSector.
}

func everyRefreshInSector(sector yacymodel.DHTRingSector) vault.KeyRange {
	return refreshOrderKeyParts.KeysWithFirst(sectorKeyOf(sector))
}

func refreshesInSectorDueBy(sector yacymodel.DHTRingSector, now time.Time) vault.KeyRange {
	return refreshOrderKeyParts.KeysWithFirstThroughSecond(sectorKeyOf(sector), now)
}

var errBadWordPartitionRecord = errors.New("bad word partition holders")

type wordPartitionRecordValueCodec struct{}

func (wordPartitionRecordValueCodec) Encode(record wordPartitionRecord) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(len(record.holders))
	for _, holder := range record.holders {
		stored.Fixed(holder.Bytes())
	}
	stored.Time(record.refreshAt)
	stored.Varint(int64(record.retryDelay))

	return stored.Record(), nil
}

func (wordPartitionRecordValueCodec) Decode(raw []byte) (wordPartitionRecord, error) {
	stored := storedfields.ReaderOf(raw, errBadWordPartitionRecord)
	holders := holdersFrom(stored)
	refreshAt := stored.Time("refresh at")
	retryDelay := time.Duration(stored.Varint("retry delay"))
	if err := stored.Err(); err != nil {
		return wordPartitionRecord{}, err
	}

	return wordPartitionRecord{holders: holders, refreshAt: refreshAt, retryDelay: retryDelay}, nil
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
