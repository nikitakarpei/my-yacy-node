package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	saturationOfHitsOfWord              = 1.2
	weightOfDocumentLength              = 0.75
	documentLengthRatioOfUnreadDocument = 1.0
)

func textScoreOf(
	foundDocument queryanswers.FoundDocument,
	queryWordRarities queryWordRarities,
	averageAmountOfWords float64,
	queryWords []yacymodel.Hash,
) float64 {
	textScore := 0.0
	for _, word := range queryWords {
		textScore += queryWordRarities.rarityShareOfWord(word) * saturatedHitsOf(
			foundDocument.Facts.HitsPerQueryWord[word],
			foundDocument.Facts.AmountOfWords,
			averageAmountOfWords,
		)
	}

	return textScore
}

func saturatedHitsOf(
	hits int,
	amountOfWords yacymodel.Optional[int],
	averageAmountOfWords float64,
) float64 {
	saturationForDocumentLength := saturationOfHitsOfWord * (1 - weightOfDocumentLength +
		weightOfDocumentLength*documentLengthRatioOf(
			amountOfWords, averageAmountOfWords,
		))

	return float64(hits) * (saturationOfHitsOfWord + 1) /
		(float64(hits) + saturationForDocumentLength)
}

func documentLengthRatioOf(
	amountOfWords yacymodel.Optional[int],
	averageAmountOfWords float64,
) float64 {
	amountOfWordsCounted, wordsCounted := amountOfWords.Get()
	if !wordsCounted || amountOfWordsCounted <= 0 || averageAmountOfWords <= 0 {
		return documentLengthRatioOfUnreadDocument
	}

	return float64(amountOfWordsCounted) / averageAmountOfWords
}
