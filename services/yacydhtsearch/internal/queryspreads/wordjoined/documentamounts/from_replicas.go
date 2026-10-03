// Package documentamounts tells the amount of documents each query word has in
// one partition of the ring: counted from the replicas of one partition, or read
// from the cache, which holds the amounts in a partition that earlier spreads
// measured.
package documentamounts

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
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

// TECHDEBT: Single source of truth — the amount of documents in a partition has two rules: complete abstracts here, reported counts in documentsperword.
func (fromReplicas FromReplicas) AmountsInAPartitionFor(
	ctx context.Context,
	query searchquery.Query,
	inquiry leadingword.Inquiry,
) map[yacymodel.Hash]int {
	askedPartition := fromReplicas.partitionToAsk(uint(fromReplicas.partitions))
	inquiry.WhichDocumentsHaveIn(askedPartition, query.WordHashes())
	inquiry.WaitUntilPartitionSettledFor(askedPartition, query.WordHashes())
	documentsPerWord := documentsperword.From(
		query, inquiry.SettledIn(askedPartition, query.WordHashes()),
	)
	countedAmounts := amountsListedIn(documentsPerWord.CompleteAbstractsIn(askedPartition))
	fromReplicas.observer.AmountsCountedFromReplicas(ctx, PerformedFromReplicas{
		Partition:                 askedPartition,
		AmountOfQueryWords:        len(query.WordHashes()),
		AmountOfQueryWordsCounted: len(countedAmounts),
	})

	return countedAmounts
}

func amountsListedIn(completeAbstracts []documentsperword.CompleteAbstract) map[yacymodel.Hash]int {
	listedAmounts := make(map[yacymodel.Hash]int, len(completeAbstracts))
	for _, completeAbstract := range completeAbstracts {
		listedAmounts[completeAbstract.Word] = len(completeAbstract.DocumentsInThePartition)
	}

	return listedAmounts
}
