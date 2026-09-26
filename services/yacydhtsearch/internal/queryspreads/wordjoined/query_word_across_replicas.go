package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordAcrossReplicas struct {
	word                 yacymodel.Hash
	replicasPerPartition [][]wordReplica
}

func queryWordsFewestDocumentsFirstFrom(
	words []yacymodel.Hash,
	settledAsks settledAsks,
	partitions yacymodel.DHTRingPartitions,
) []queryWordAcrossReplicas {
	queryWordsAcrossReplicas := queryWordsAcrossReplicasFrom(words, settledAsks, partitions)
	slices.SortStableFunc(queryWordsAcrossReplicas, fewestDocumentsFirst)

	return queryWordsAcrossReplicas
}

func queryWordsAcrossReplicasFrom(
	words []yacymodel.Hash,
	settledAsks settledAsks,
	partitions yacymodel.DHTRingPartitions,
) []queryWordAcrossReplicas {
	queryWordsAcrossReplicas := make([]queryWordAcrossReplicas, 0, len(words))
	for _, word := range words {
		queryWordsAcrossReplicas = append(
			queryWordsAcrossReplicas,
			queryWordAcrossReplicasFrom(word, settledAsks, partitions),
		)
	}

	return queryWordsAcrossReplicas
}

func queryWordAcrossReplicasFrom(
	word yacymodel.Hash,
	settledAsks settledAsks,
	partitions yacymodel.DHTRingPartitions,
) queryWordAcrossReplicas {
	replicasPerPartition := make([][]wordReplica, partitions)
	for _, settledAsk := range settledAsks {
		if settledAsk.Word != word {
			continue
		}
		for _, answer := range settledAsk.Answers {
			replicasPerPartition[settledAsk.Partition] = append(
				replicasPerPartition[settledAsk.Partition],
				wordReplica{
					listedDocuments:  answer.ListedDocuments,
					documentsToMatch: settledAsk.DocumentsToMatch,
				},
			)
		}
	}

	return queryWordAcrossReplicas{word: word, replicasPerPartition: replicasPerPartition}
}

func fewestDocumentsFirst(first, second queryWordAcrossReplicas) int {
	documentsOfTheFirst, countedForTheFirst := first.estimatedAmountOfDocumentsHeld().Get()
	documentsOfTheSecond, countedForTheSecond := second.estimatedAmountOfDocumentsHeld().Get()
	if countedForTheFirst != countedForTheSecond {
		if countedForTheFirst {
			return -1
		}

		return 1
	}

	return cmp.Compare(documentsOfTheFirst, documentsOfTheSecond)
}

func (queryWord queryWordAcrossReplicas) estimatedAmountOfDocumentsHeld() yacymodel.Optional[int] {
	amountsHeldInCompletePartitions := queryWord.amountsHeldInCompletePartitions()
	if len(amountsHeldInCompletePartitions) == 0 {
		return yacymodel.None[int]()
	}

	amountOfIncompletePartitions := len(queryWord.replicasPerPartition) -
		len(amountsHeldInCompletePartitions)
	sumOfAmountsHeld := amountOfIncompletePartitions *
		lowerMedianOf(amountsHeldInCompletePartitions)
	for _, amountHeld := range amountsHeldInCompletePartitions {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (queryWord queryWordAcrossReplicas) amountsHeldInCompletePartitions() []int {
	amountsHeld := make([]int, 0, len(queryWord.replicasPerPartition))
	for _, replicasOfPartition := range queryWord.replicasPerPartition {
		amountsHeldInThePartition := amountsHeldBy(replicasOfPartition)
		if len(amountsHeldInThePartition) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(amountsHeldInThePartition))
	}

	return amountsHeld
}

func amountsHeldBy(replicas []wordReplica) []int {
	amountsHeld := make([]int, 0, len(replicas))
	for _, replica := range replicas {
		if !replica.hasACompleteAbstract() {
			continue
		}
		amountsHeld = append(amountsHeld, len(replica.listedDocuments))
	}

	return amountsHeld
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (queryWord queryWordAcrossReplicas) documents() distinctDocuments {
	documents := distinctDocuments{}
	for _, replicasOfPartition := range queryWord.replicasPerPartition {
		for _, replica := range replicasOfPartition {
			for _, listedDocument := range replica.listedDocuments {
				documents.add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (queryWord queryWordAcrossReplicas) sampleIn(
	partition uint,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[int] {
	documentsInThePartition := distinctDocuments{}
	complete := false
	for _, replica := range queryWord.replicasPerPartition[partition] {
		if !replica.hasACompleteAbstract() {
			continue
		}
		complete = true
		for _, listedDocument := range replica.listedDocuments {
			if partitions.PartitionOf(listedDocument.Hash) != partition {
				continue
			}
			documentsInThePartition.add(listedDocument.Hash)
		}
	}
	if !complete {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(len(documentsInThePartition))
}
