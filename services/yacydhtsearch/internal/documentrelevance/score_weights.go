package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ScoreWeights struct {
	WeightOfTheTitleScore        float64
	WeightOfTheTextScore         float64
	WeightOfTheAddressScore      float64
	WeightOfThePhraseScore       float64
	WeightOfTheCoordinationScore float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		WeightOfTheTitleScore:        10.0,
		WeightOfTheTextScore:         0.25,
		WeightOfTheAddressScore:      1.0,
		WeightOfThePhraseScore:       3.0,
		WeightOfTheCoordinationScore: 0.5,
	}
}

func (weights ScoreWeights) relevanceOf(
	item queryanswers.FoundDocument,
	rarityOfTheQueryWords queryWordRarity,
	averageDocumentLength float64,
	queryWords []yacymodel.Hash,
) float64 {
	return weights.WeightOfTheTitleScore*titleScoreOf(item, rarityOfTheQueryWords, queryWords) +
		weights.WeightOfTheTextScore*
			textScoreOf(item, rarityOfTheQueryWords, averageDocumentLength, queryWords) +
		weights.WeightOfTheAddressScore*addressScoreOf(item, queryWords) +
		weights.WeightOfThePhraseScore*phraseScoreOf(item) +
		weights.WeightOfTheCoordinationScore*coordinationScoreOf(item, queryWords)
}
