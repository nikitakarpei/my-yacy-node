package documentamounts

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type answersOfWord []answersOfPartition

func (answers answersOfWord) estimatedAmountHeld() yacymodel.Optional[int] {
	countedAmounts := answers.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}
	amountOfPartitionsWhereNoPeerCounted := len(answers) - len(countedAmounts)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted * lowerMedianOf(countedAmounts)
	for _, countedAmount := range countedAmounts {
		sumOfAmountsHeld += countedAmount
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (answers answersOfWord) countedAmounts() []int {
	countedAmounts := make([]int, 0, len(answers))
	for _, answersOfThePartition := range answers {
		if countedAmount, counted := answersOfThePartition.countedAmount().Get(); counted {
			countedAmounts = append(countedAmounts, countedAmount)
		}
	}

	return countedAmounts
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (answers answersOfWord) amountInAPartition() yacymodel.Optional[int] {
	countedAmounts := answers.countedAmounts()
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(lowerMedianOf(countedAmounts))
}

func (answers answersOfWord) listAnyDocument() bool {
	for _, answersOfThePartition := range answers {
		for _, answer := range answersOfThePartition {
			if len(answer.ListedDocuments) > 0 {
				return true
			}
		}
	}

	return false
}

func (answers answersOfWord) documents() yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, answersOfThePartition := range answers {
		for _, answer := range answersOfThePartition {
			for _, listedDocument := range answer.ListedDocuments {
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

type answersOfPartition []wordpartitionasks.ReplicaAnswer

func (answers answersOfPartition) countedAmount() yacymodel.Optional[int] {
	countedAmounts := make([]int, 0, len(answers))
	for _, answer := range answers {
		if amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get(); counted {
			countedAmounts = append(countedAmounts, amountOfDocumentsHeld)
		}
	}
	if len(countedAmounts) == 0 {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(lowerMedianOf(countedAmounts))
}
