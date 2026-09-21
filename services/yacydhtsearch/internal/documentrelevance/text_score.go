package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	saturationOfTheHitsOfAWord                    = 1.2
	weightOfTheDocumentLength                     = 0.75
	lengthRatioOfADocumentNobodyCountedTheWordsOf = 1.0
)

func textScoreOf(
	facts queryanswers.DocumentFacts,
	rarity queryWordRarity,
	averageAmountOfWordsAnyoneCounted float64,
	queryWords []yacymodel.Hash,
) float64 {
	sumOfTheSaturatedHits := 0.0
	for _, word := range queryWords {
		sumOfTheSaturatedHits += rarity.rarityOfTheQueryWord(word) * saturatedHitsOf(
			facts.HitsPerQueryWord[word],
			facts.AmountOfWords,
			averageAmountOfWordsAnyoneCounted,
		)
	}

	return sumOfTheSaturatedHits / rarity.sumOfTheRarityOfTheQueryWords
}

func averageAmountOfWordsAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) float64 {
	sumOfTheAmountsOfWords, amountOfCountedDocuments := 0, 0
	for _, facts := range factsPerDocument {
		amountOfWords, counted := facts.AmountOfWords.Get()
		if !counted || amountOfWords <= 0 {
			continue
		}
		sumOfTheAmountsOfWords += amountOfWords
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return 0
	}

	return float64(sumOfTheAmountsOfWords) / float64(amountOfCountedDocuments)
}

func saturatedHitsOf(
	hits int,
	amountOfWords yacymodel.Optional[int],
	averageAmountOfWordsAnyoneCounted float64,
) float64 {
	countedHits := float64(hits)
	saturationForTheDocumentLength := saturationOfTheHitsOfAWord * (1 - weightOfTheDocumentLength +
		weightOfTheDocumentLength*documentLengthRatioOf(
			amountOfWords, averageAmountOfWordsAnyoneCounted,
		))

	return countedHits * (saturationOfTheHitsOfAWord + 1) /
		(countedHits + saturationForTheDocumentLength)
}

func documentLengthRatioOf(
	amountOfWords yacymodel.Optional[int],
	averageAmountOfWordsAnyoneCounted float64,
) float64 {
	countedAmountOfWords, counted := amountOfWords.Get()
	if !counted || countedAmountOfWords <= 0 || averageAmountOfWordsAnyoneCounted <= 0 {
		return lengthRatioOfADocumentNobodyCountedTheWordsOf
	}

	return float64(countedAmountOfWords) / averageAmountOfWordsAnyoneCounted
}
