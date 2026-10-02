package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordAcrossReplicas struct {
	hash                yacymodel.Hash
	answersPerPartition [][]wordpartitionasks.ReplicaAnswer
}

func (word wordAcrossReplicas) withAnswersOf(
	settledAsk wordpartitionasks.SettledAsk,
) wordAcrossReplicas {
	answersPerPartition := slices.Clone(word.answersPerPartition)
	answersPerPartition[settledAsk.Partition] = append(
		slices.Clip(answersPerPartition[settledAsk.Partition]),
		settledAsk.Answers...,
	)

	return wordAcrossReplicas{hash: word.hash, answersPerPartition: answersPerPartition}
}

func fewestDocumentsFirst(first, second wordAcrossReplicas) int {
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

func (word wordAcrossReplicas) estimatedAmountOfDocumentsHeld() yacymodel.Optional[int] {
	amountsHeldInPartitionsWhereAPeerCounted := word.amountsOfDocumentsHeldInPartitionsWhereAPeerCounted()
	if len(amountsHeldInPartitionsWhereAPeerCounted) == 0 {
		return yacymodel.None[int]()
	}

	amountOfPartitionsWhereNoPeerCounted := len(word.answersPerPartition) -
		len(amountsHeldInPartitionsWhereAPeerCounted)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted *
		lowerMedianOf(amountsHeldInPartitionsWhereAPeerCounted)
	for _, amountHeld := range amountsHeldInPartitionsWhereAPeerCounted {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

// TECHDEBT: Naming — a name past four words names two facts: amountsOfDocumentsHeldInPartitionsWhereAPeerCounted.
func (word wordAcrossReplicas) amountsOfDocumentsHeldInPartitionsWhereAPeerCounted() []int {
	amountsHeld := make([]int, 0, len(word.answersPerPartition))
	for _, answersOfPartition := range word.answersPerPartition {
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

func (word wordAcrossReplicas) documents() distinctDocuments {
	documents := distinctDocuments{}
	for _, answersOfPartition := range word.answersPerPartition {
		for _, answer := range answersOfPartition {
			for _, listedDocument := range answer.ListedDocuments {
				documents.add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (word wordAcrossReplicas) sampleIn(
	partition uint,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.Optional[int] {
	documentsInThePartition := distinctDocuments{}
	complete := false
	for _, answer := range word.answersPerPartition[partition] {
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

func (word wordAcrossReplicas) everyAnswer() []wordpartitionasks.ReplicaAnswer {
	var everyAnswer []wordpartitionasks.ReplicaAnswer
	for _, answersOfPartition := range word.answersPerPartition {
		everyAnswer = append(everyAnswer, answersOfPartition...)
	}

	return everyAnswer
}
