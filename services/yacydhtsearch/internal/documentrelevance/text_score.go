package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	saturationOfTheHitsOfAWord                = 1.2
	weightOfTheDocumentLength                 = 0.75
	lengthRatioOfADocumentWhosePageNoNodeRead = 1.0
)

func textScoreOf(
	foundDocument queryanswers.FoundDocument,
	rarity queryWordRarity,
	averageLengthOfTheReadPages float64,
	queryWords []yacymodel.Hash,
) float64 {
	textScore := 0.0
	for _, word := range queryWords {
		textScore += rarity.rarityOfTheQueryWord(word) * saturatedHitsOf(
			foundDocument.HitsPerQueryWord[word],
			foundDocument.AmountOfWordsOfTheReadPage,
			averageLengthOfTheReadPages,
		)
	}

	return textScore
}

func averageLengthOfTheReadPagesAmong(foundDocuments []queryanswers.FoundDocument) float64 {
	sumOfTheAmountsOfWords, amountOfReadPages := 0, 0
	for _, foundDocument := range foundDocuments {
		amountOfWords, read := foundDocument.AmountOfWordsOfTheReadPage.Get()
		if !read || amountOfWords <= 0 {
			continue
		}
		sumOfTheAmountsOfWords += amountOfWords
		amountOfReadPages++
	}
	if amountOfReadPages == 0 {
		return 0
	}

	return float64(sumOfTheAmountsOfWords) / float64(amountOfReadPages)
}

func saturatedHitsOf(
	hits int,
	amountOfWordsOfTheReadPage yacymodel.Optional[int],
	averageLengthOfTheReadPages float64,
) float64 {
	countedHits := float64(hits)
	saturationForTheDocumentLength := saturationOfTheHitsOfAWord * (1 - weightOfTheDocumentLength +
		weightOfTheDocumentLength*documentLengthRatioOf(
			amountOfWordsOfTheReadPage, averageLengthOfTheReadPages,
		))

	return countedHits * (saturationOfTheHitsOfAWord + 1) /
		(countedHits + saturationForTheDocumentLength)
}

func documentLengthRatioOf(
	amountOfWordsOfTheReadPage yacymodel.Optional[int],
	averageLengthOfTheReadPages float64,
) float64 {
	amountOfWords, read := amountOfWordsOfTheReadPage.Get()
	if !read || amountOfWords <= 0 || averageLengthOfTheReadPages <= 0 {
		return lengthRatioOfADocumentWhosePageNoNodeRead
	}

	return float64(amountOfWords) / averageLengthOfTheReadPages
}
