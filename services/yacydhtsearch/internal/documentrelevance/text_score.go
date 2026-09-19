package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	saturationOfTheHitsOfAWord           = 1.2
	weightOfTheDocumentLength            = 0.75
	lengthRatioOfADocumentNoPeerMeasured = 1.0
)

func textScoreOf(
	foundDocument queryanswers.FoundDocument,
	rarity queryWordRarity,
	averageDocumentLength float64,
	queryWords []yacymodel.Hash,
) float64 {
	textScore := 0.0
	for _, word := range queryWords {
		textScore += rarity.rarityOfTheQueryWord(word) * saturatedHitsOf(
			foundDocument.HitsPerQueryWord[word],
			foundDocument.AmountOfWords,
			averageDocumentLength,
		)
	}

	return textScore
}

func averageDocumentLengthOf(foundDocuments []queryanswers.FoundDocument) float64 {
	sumOfTheAmountsOfWords, amountOfMeasuredDocuments := 0, 0
	for _, foundDocument := range foundDocuments {
		if foundDocument.AmountOfWords <= 0 {
			continue
		}
		sumOfTheAmountsOfWords += foundDocument.AmountOfWords
		amountOfMeasuredDocuments++
	}
	if amountOfMeasuredDocuments == 0 {
		return 0
	}

	return float64(sumOfTheAmountsOfWords) / float64(amountOfMeasuredDocuments)
}

func saturatedHitsOf(hits int, amountOfWords int, averageDocumentLength float64) float64 {
	countedHits := float64(hits)
	saturationForTheDocumentLength := saturationOfTheHitsOfAWord * (1 - weightOfTheDocumentLength +
		weightOfTheDocumentLength*documentLengthRatioOf(amountOfWords, averageDocumentLength))

	return countedHits * (saturationOfTheHitsOfAWord + 1) /
		(countedHits + saturationForTheDocumentLength)
}

func documentLengthRatioOf(amountOfWords int, averageDocumentLength float64) float64 {
	if amountOfWords <= 0 || averageDocumentLength <= 0 {
		return lengthRatioOfADocumentNoPeerMeasured
	}

	return float64(amountOfWords) / averageDocumentLength
}
