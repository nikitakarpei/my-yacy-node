package wordholdings

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type holdingsOfWord struct {
	word                yacymodel.Hash
	partitions          yacymodel.DHTRingPartitions
	answersPerPartition []answersOfPartition
}

func holdingsOfEachWordFrom(
	words []yacymodel.Hash,
	settledAsks []wordpartitionasks.SettledAsk,
	partitions yacymodel.DHTRingPartitions,
) []holdingsOfWord {
	holdingsOfEachWord := make([]holdingsOfWord, 0, len(words))
	for _, word := range words {
		holdingsOfEachWord = append(
			holdingsOfEachWord, holdingsOfWordFrom(word, settledAsks, partitions),
		)
	}

	return holdingsOfEachWord
}

func holdingsOfWordFrom(
	word yacymodel.Hash,
	settledAsks []wordpartitionasks.SettledAsk,
	partitions yacymodel.DHTRingPartitions,
) holdingsOfWord {
	answersPerPartition := make([]answersOfPartition, partitions)
	for _, settledAsk := range settledAsks {
		if settledAsk.Word != word {
			continue
		}
		answersPerPartition[settledAsk.Partition] = append(
			answersPerPartition[settledAsk.Partition], settledAsk.Answers...,
		)
	}

	return holdingsOfWord{
		word:                word,
		partitions:          partitions,
		answersPerPartition: answersPerPartition,
	}
}

func (holdings holdingsOfWord) documents() yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, answersOfPartition := range holdings.answersPerPartition {
		for _, answer := range answersOfPartition {
			for _, listedDocument := range answer.ListedDocuments {
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (holdings holdingsOfWord) estimatedAmountOfDocumentsHeld() yacymodel.Optional[int] {
	countedAmounts := holdings.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	amountOfPartitionsWhereNoPeerCounted := len(holdings.answersPerPartition) - len(countedAmounts)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted * lowerMedianOf(countedAmounts)
	for _, countedAmount := range countedAmounts {
		sumOfAmountsHeld += countedAmount
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (holdings holdingsOfWord) amountOfDocumentsInAPartition() yacymodel.Optional[int] {
	countedAmounts := holdings.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(lowerMedianOf(countedAmounts))
}

func (holdings holdingsOfWord) countedAmounts() []int {
	countedAmounts := make([]int, 0, len(holdings.answersPerPartition))
	for _, answers := range holdings.answersPerPartition {
		if countedAmount, counted := answers.countedAmount().Get(); counted {
			countedAmounts = append(countedAmounts, countedAmount)
		}
	}

	return countedAmounts
}

type answersOfPartition []wordpartitionasks.ReplicaAnswer

func (answers answersOfPartition) countedAmount() yacymodel.Optional[int] {
	countedAmounts := make([]int, 0, len(answers))
	for _, answer := range answers {
		amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get()
		if !counted {
			continue
		}
		countedAmounts = append(countedAmounts, amountOfDocumentsHeld)
	}
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(lowerMedianOf(countedAmounts))
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (holdings holdingsOfWord) completeAbstractIn(
	partition uint,
) yacymodel.Optional[yacymodel.URLHashes] {
	documentsInThePartition := yacymodel.URLHashes{}
	complete := false
	for _, answer := range holdings.answersPerPartition[partition] {
		if !abstractIsComplete(answer) {
			continue
		}
		complete = true
		for _, listedDocument := range answer.ListedDocuments {
			if holdings.partitions.PartitionOf(listedDocument.Hash) != partition {
				continue
			}
			documentsInThePartition.Add(listedDocument.Hash)
		}
	}
	if !complete {
		return yacymodel.None[yacymodel.URLHashes]()
	}

	return yacymodel.Some(documentsInThePartition)
}

func abstractIsComplete(answer wordpartitionasks.ReplicaAnswer) bool {
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get()
	if !counted {
		return answer.Searched && len(answer.ListedDocuments) == 0
	}

	return amountOfDocumentsHeld <= len(answer.ListedDocuments)
}

func fewestDocumentsFirst(first, second holdingsOfWord) int {
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
