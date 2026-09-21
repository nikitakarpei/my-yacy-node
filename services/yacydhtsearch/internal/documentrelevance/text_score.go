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
	rarityOfTheQuery queryRarity,
	averageAmountOfWordsAnyoneCounted float64,
	queryWords []yacymodel.Hash,
) float64 {
	textScore := 0.0
	for _, word := range queryWords {
		textScore += rarityOfTheQuery.shareHeldByTheQueryWord(word) * saturatedHitsOf(
			facts.HitsPerQueryWord[word],
			facts.AmountOfWords,
			averageAmountOfWordsAnyoneCounted,
		)
	}

	return textScore
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
