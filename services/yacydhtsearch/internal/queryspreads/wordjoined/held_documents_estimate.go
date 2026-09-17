package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func amountOfDocumentsHeldPerQueryWordOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	peersPerQueryWord []peerchoice.PeersOfQueryWord,
	queryWords []yacymodel.Hash,
) map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(queryWords))
	for index, queryWord := range queryWords {
		countedAmountsPerPartition := countedAmountsOfDocumentsHeldPerPartitionOf(
			answeredAsks, peersPerQueryWord[index], queryWord,
		)
		amountOfDocumentsHeld, counted := amountOfDocumentsHeldAcrossPartitionsOf(
			countedAmountsPerPartition,
		)
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

func countedAmountsOfDocumentsHeldPerPartitionOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	peersOfQueryWord peerchoice.PeersOfQueryWord,
	queryWord yacymodel.Hash,
) [][]int {
	partitionOfEachPeer := partitionOfEachPeerOf(peersOfQueryWord)
	countedAmountsPerPartition := make([][]int, peersOfQueryWord.Partitions)
	for _, answeredAsk := range answeredAsks {
		if answeredAsk.Ask.Word != queryWord {
			continue
		}
		amountOfDocumentsHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		partition, chosen := partitionOfEachPeer[answeredAsk.Ask.Peer.Hash]
		if !chosen {
			continue
		}
		countedAmountsPerPartition[partition] = append(
			countedAmountsPerPartition[partition], amountOfDocumentsHeld,
		)
	}

	return countedAmountsPerPartition
}

func partitionOfEachPeerOf(
	peersOfQueryWord peerchoice.PeersOfQueryWord,
) map[yacymodel.Hash]uint {
	partitionOfEachPeer := make(map[yacymodel.Hash]uint, len(peersOfQueryWord.ChosenPeers))
	for _, chosenPeer := range peersOfQueryWord.ChosenPeers {
		partitionOfEachPeer[chosenPeer.Peer.Hash] = chosenPeer.Partition
	}

	return partitionOfEachPeer
}

func amountOfDocumentsHeldAcrossPartitionsOf(countedAmountsPerPartition [][]int) (int, bool) {
	amountsHeldInCountedPartitions := amountsOfDocumentsHeldInCountedPartitionsOf(
		countedAmountsPerPartition,
	)
	if len(amountsHeldInCountedPartitions) == 0 {
		return 0, false
	}

	amountOfUncountedPartitions := len(countedAmountsPerPartition) -
		len(amountsHeldInCountedPartitions)
	sumOfAmountsHeld := amountOfUncountedPartitions * lowerMedianOf(amountsHeldInCountedPartitions)
	for _, amountHeld := range amountsHeldInCountedPartitions {
		sumOfAmountsHeld += amountHeld
	}

	return sumOfAmountsHeld, true
}

func amountsOfDocumentsHeldInCountedPartitionsOf(countedAmountsPerPartition [][]int) []int {
	amountsHeld := make([]int, 0, len(countedAmountsPerPartition))
	for _, countedAmounts := range countedAmountsPerPartition {
		if len(countedAmounts) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(countedAmounts))
	}

	return amountsHeld
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}
