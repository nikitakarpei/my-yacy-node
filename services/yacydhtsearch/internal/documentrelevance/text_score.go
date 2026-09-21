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

type textScorer struct {
	queryWordRarities    queryWordRarities
	queryWords           []yacymodel.Hash
	averageAmountOfWords float64
}

func textScorerFrom(statistics answersStatistics) textScorer {
	return textScorer{
		queryWordRarities:    statistics.queryWordRarities,
		queryWords:           statistics.queryWords,
		averageAmountOfWords: statistics.documentAverages.averageAmountOfWords,
	}
}

func (scorer textScorer) scoreOf(document queryanswers.FoundDocument) float64 {
	textScore := 0.0
	for _, word := range scorer.queryWords {
		textScore += scorer.queryWordRarities.rarityShareOfWord(word) *
			scorer.saturatedHitsOfWordIn(word, document)
	}

	return textScore
}

func (scorer textScorer) saturatedHitsOfWordIn(
	word yacymodel.Hash, document queryanswers.FoundDocument,
) float64 {
	hitsOfWord := float64(document.Facts.HitsPerQueryWord[word])
	saturationForDocumentLength := saturationOfHitsOfWord * (1 - weightOfDocumentLength +
		weightOfDocumentLength*scorer.documentLengthRatioOf(document.Facts.AmountOfWords))

	return hitsOfWord * (saturationOfHitsOfWord + 1) /
		(hitsOfWord + saturationForDocumentLength)
}

func (scorer textScorer) documentLengthRatioOf(
	amountOfWords yacymodel.Optional[int],
) float64 {
	amountOfWordsCounted, wordsCounted := amountOfWords.Get()
	if !wordsCounted || amountOfWordsCounted <= 0 || scorer.averageAmountOfWords <= 0 {
		return documentLengthRatioOfUnreadDocument
	}

	return float64(amountOfWordsCounted) / scorer.averageAmountOfWords
}
