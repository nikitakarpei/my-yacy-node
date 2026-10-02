package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordAcrossReplicas struct {
	word                yacymodel.Hash
	answersPerPartition [][]wordpartitionasks.ReplicaAnswer
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
	answersPerPartition := make([][]wordpartitionasks.ReplicaAnswer, partitions)
	for _, settledAsk := range settledAsks {
		if settledAsk.Word != word {
			continue
		}
		answersPerPartition[settledAsk.Partition] = append(
			answersPerPartition[settledAsk.Partition],
			settledAsk.Answers...,
		)
	}

	return queryWordAcrossReplicas{word: word, answersPerPartition: answersPerPartition}
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
	amountsHeldInPartitionsWhereAPeerCounted := queryWord.amountsOfDocumentsHeldInPartitionsWhereAPeerCounted()
	if len(amountsHeldInPartitionsWhereAPeerCounted) == 0 {
		return yacymodel.None[int]()
	}

	amountOfPartitionsWhereNoPeerCounted := len(queryWord.answersPerPartition) -
		len(amountsHeldInPartitionsWhereAPeerCounted)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted *
		lowerMedianOf(amountsHeldInPartitionsWhereAPeerCounted)
	for _, amountHeld := range amountsHeldInPartitionsWhereAPeerCounted {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

// TECHDEBT: Naming — a name past four words names two facts: amountsOfDocumentsHeldInPartitionsWhereAPeerCounted.
func (queryWord queryWordAcrossReplicas) amountsOfDocumentsHeldInPartitionsWhereAPeerCounted() []int {
	amountsHeld := make([]int, 0, len(queryWord.answersPerPartition))
	for _, answersOfPartition := range queryWord.answersPerPartition {
		countedAmounts := amountsOfDocumentsHeldCountedBy(answersOfPartition)
		if len(countedAmounts) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(countedAmounts))
	}

	return amountsHeld
}

func amountsOfDocumentsHeldCountedBy(answers []wordpartitionasks.ReplicaAnswer) []int {
	countedAmounts := make([]int, 0, len(answers))
	for _, answer := range answers {
		amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get()
		if !counted {
			continue
		}
		countedAmounts = append(countedAmounts, amountOfDocumentsHeld)
	}

	return countedAmounts
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (queryWord queryWordAcrossReplicas) documents() distinctDocuments {
	documents := distinctDocuments{}
	for _, answersOfPartition := range queryWord.answersPerPartition {
		for _, answer := range answersOfPartition {
			for _, listedDocument := range answer.ListedDocuments {
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
	for _, answer := range queryWord.answersPerPartition[partition] {
		if !answer.HasACompleteAbstract() {
			continue
		}
		complete = true
		for _, listedDocument := range answer.ListedDocuments {
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
