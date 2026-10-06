package documentsperword

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsOfWord struct {
	word                yacymodel.Hash
	partitions          yacymodel.DHTRingPartitions
	answersPerPartition []answersOfPartition
}

func documentsOfEachWordFrom(
	words []yacymodel.Hash,
	answeredWordPartitions []documentasks.AnsweredWordPartition,
	partitions yacymodel.DHTRingPartitions,
) []documentsOfWord {
	documentsOfEachWord := make([]documentsOfWord, 0, len(words))
	for _, word := range words {
		documentsOfEachWord = append(
			documentsOfEachWord, documentsOfWordFrom(word, answeredWordPartitions, partitions),
		)
	}

	return documentsOfEachWord
}

func documentsOfWordFrom(
	word yacymodel.Hash,
	answeredWordPartitions []documentasks.AnsweredWordPartition,
	partitions yacymodel.DHTRingPartitions,
) documentsOfWord {
	answersPerPartition := make([]answersOfPartition, partitions)
	for _, answered := range answeredWordPartitions {
		if answered.Word != word {
			continue
		}
		answersPerPartition[answered.Partition] = append(
			answersPerPartition[answered.Partition], answered.Answers...,
		)
	}

	return documentsOfWord{
		word:                word,
		partitions:          partitions,
		answersPerPartition: answersPerPartition,
	}
}

func (wordDocuments documentsOfWord) documents() yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, answersOfPartition := range wordDocuments.answersPerPartition {
		for _, answer := range answersOfPartition {
			for _, listedDocument := range answer.ListedDocuments {
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (wordDocuments documentsOfWord) estimatedAmountOfDocumentsHeld() yacymodel.Optional[int] {
	countedAmounts := wordDocuments.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	amountOfPartitionsWhereNoPeerCounted := len(
		wordDocuments.answersPerPartition,
	) - len(
		countedAmounts,
	)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted * lowerMedianOf(countedAmounts)
	for _, countedAmount := range countedAmounts {
		sumOfAmountsHeld += countedAmount
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (wordDocuments documentsOfWord) amountOfDocumentsInAPartition() yacymodel.Optional[int] {
	countedAmounts := wordDocuments.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(lowerMedianOf(countedAmounts))
}

func (wordDocuments documentsOfWord) countedAmounts() []int {
	countedAmounts := make([]int, 0, len(wordDocuments.answersPerPartition))
	for _, answers := range wordDocuments.answersPerPartition {
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

func (wordDocuments documentsOfWord) completeAbstractIn(
	partition uint,
) yacymodel.Optional[yacymodel.URLHashes] {
	documentsInThePartition := yacymodel.URLHashes{}
	complete := false
	for _, answer := range wordDocuments.answersPerPartition[partition] {
		if !abstractIsComplete(answer) {
			continue
		}
		complete = true
		for _, listedDocument := range answer.ListedDocuments {
			if wordDocuments.partitions.PartitionOf(listedDocument.Hash) != partition {
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

func fewestDocumentsFirst(first, second documentsOfWord) int {
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
