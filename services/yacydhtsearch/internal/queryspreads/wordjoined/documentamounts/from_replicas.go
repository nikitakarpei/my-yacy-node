// Package documentamounts tells the amount of documents each query word has. Before
// the inquiry, it tells the amount in one partition of the ring: counted from the
// replicas of one partition, or read from the cache of earlier spreads. Its
// measurement of the inquiry tells, from every answer, the amount held in the whole
// network and in a partition.
package documentamounts

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
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

// TECHDEBT: Single source of truth — the amount of documents in a partition has two rules: complete abstracts here, reported counts in Measurement.
func (fromReplicas FromReplicas) AmountsInAPartitionFor(
	ctx context.Context,
	query searchquery.Query,
	documentAsks leadingword.DocumentAsks,
) map[yacymodel.Hash]int {
	askedPartition := fromReplicas.partitionToAsk(uint(fromReplicas.partitions))
	countedAmounts := fromReplicas.amountsInTheCompleteAbstractsOf(
		askedPartition,
		documentAsks.WhichDocumentsHaveIn(askedPartition, query.WordHashes()),
	)
	fromReplicas.observer.AmountsCountedFromReplicas(ctx, PerformedFromReplicas{
		Partition:                 askedPartition,
		AmountOfQueryWords:        len(query.WordHashes()),
		AmountOfQueryWordsCounted: len(countedAmounts),
	})

	return countedAmounts
}

func (fromReplicas FromReplicas) amountsInTheCompleteAbstractsOf(
	partition uint,
	answered []wordpartitionasks.SettledAsk,
) map[yacymodel.Hash]int {
	documentsPerWord := map[yacymodel.Hash]yacymodel.URLHashes{}
	for _, settledAsk := range answered {
		for _, answer := range settledAsk.Answers {
			if !abstractIsComplete(answer) {
				continue
			}
			if documentsPerWord[settledAsk.Word] == nil {
				documentsPerWord[settledAsk.Word] = yacymodel.URLHashes{}
			}
			fromReplicas.keepTheDocumentsIn(partition, answer, documentsPerWord[settledAsk.Word])
		}
	}
	amountsPerWord := make(map[yacymodel.Hash]int, len(documentsPerWord))
	for word, documents := range documentsPerWord {
		amountsPerWord[word] = len(documents)
	}

	return amountsPerWord
}

func abstractIsComplete(answer wordpartitionasks.ReplicaAnswer) bool {
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get()
	if !counted {
		return answer.Searched && len(answer.ListedDocuments) == 0
	}

	return amountOfDocumentsHeld <= len(answer.ListedDocuments)
}

func (fromReplicas FromReplicas) keepTheDocumentsIn(
	partition uint,
	answer wordpartitionasks.ReplicaAnswer,
	documents yacymodel.URLHashes,
) {
	for _, listedDocument := range answer.ListedDocuments {
		if fromReplicas.partitions.PartitionOf(listedDocument.Hash) == partition {
			documents.Add(listedDocument.Hash)
		}
	}
}
