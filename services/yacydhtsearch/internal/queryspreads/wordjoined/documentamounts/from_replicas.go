// Package documentamounts tells the amount of documents each query word has in
// one partition of the ring: counted from the replicas of one partition, or read
// from the cache, which holds the amounts in a partition that earlier spreads
// measured.
package documentamounts

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordholdings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FromReplicas struct {
	partitions     yacymodel.DHTRingPartitions
	partitionToAsk func(amountOfPartitions uint) uint
	observer       FromReplicasObserver
}

func NewFromReplicas(
	partitions yacymodel.DHTRingPartitions,
	partitionToAsk func(amountOfPartitions uint) uint,
	observer FromReplicasObserver,
) FromReplicas {
	return FromReplicas{partitions: partitions, partitionToAsk: partitionToAsk, observer: observer}
}

// TECHDEBT: Single source of truth — the amount of documents in a partition has two rules: complete abstracts here, reported counts in wordholdings.
func (fromReplicas FromReplicas) AmountsInAPartitionFor(
	ctx context.Context,
	query searchquery.Query,
	run leadingword.WordAsks,
) map[yacymodel.Hash]int {
	askedPartition := fromReplicas.partitionToAsk(uint(fromReplicas.partitions))
	run.AskPartitionFor(askedPartition, query.WordHashes())
	run.WaitUntilPartitionSettledFor(askedPartition, query.WordHashes())
	holdings := wordholdings.OfEachQueryWord(
		query, run.SettledIn(askedPartition, query.WordHashes()), fromReplicas.partitions,
	)
	countedAmounts := amountsListedIn(holdings.CompleteAbstractsIn(askedPartition))
	fromReplicas.observer.AmountsCountedFromReplicas(ctx, PerformedFromReplicas{
		Partition:                 askedPartition,
		AmountOfQueryWords:        len(query.WordHashes()),
		AmountOfQueryWordsCounted: len(countedAmounts),
	})

	return countedAmounts
}

func amountsListedIn(completeAbstracts []wordholdings.CompleteAbstract) map[yacymodel.Hash]int {
	listedAmounts := make(map[yacymodel.Hash]int, len(completeAbstracts))
	for _, completeAbstract := range completeAbstracts {
		listedAmounts[completeAbstract.Word] = len(completeAbstract.DocumentsInThePartition)
	}

	return listedAmounts
}
