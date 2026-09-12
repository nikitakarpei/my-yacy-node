package relevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ScoreWeights struct {
	WeightOfThePlaceScore        float64
	WeightOfTheTitleScore        float64
	WeightOfTheTextScore         float64
	WeightOfTheAddressScore      float64
	WeightOfThePhraseScore       float64
	WeightOfTheCoordinationScore float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		WeightOfThePlaceScore:        0.25,
		WeightOfTheTitleScore:        3.0,
		WeightOfTheTextScore:         0.25,
		WeightOfTheAddressScore:      1.0,
		WeightOfThePhraseScore:       3.0,
		WeightOfTheCoordinationScore: 0.5,
	}
}

func (weights ScoreWeights) relevanceOf(
	item peeranswers.AnsweredItem,
	placeScore float64,
	rarityPerQueryWord map[yacymodel.Hash]float64,
	averageDocumentLength float64,
	amountOfQueryWords int,
) float64 {
	return weights.WeightOfThePlaceScore*placeScore +
		weights.WeightOfTheTitleScore*titleScoreOf(item) +
		weights.WeightOfTheTextScore*
			textScoreOf(item, rarityPerQueryWord, averageDocumentLength) +
		weights.WeightOfTheAddressScore*addressScoreOf(item) +
		weights.WeightOfThePhraseScore*phraseScoreOf(item) +
		weights.WeightOfTheCoordinationScore*coordinationScoreOf(item, amountOfQueryWords)
}
