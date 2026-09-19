package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ScoreWeights struct {
	WeightOfTheTitleScore          float64
	WeightOfTheTextScore           float64
	WeightOfTheAddressScore        float64
	WeightOfThePhraseScore         float64
	WeightOfTheCoordinationScore   float64
	WeightOfTheNamedSiteEntryScore float64
	WeightOfTheOverlongTextScore   float64
	WeightOfTheLinkSparsityPenalty float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		WeightOfTheTitleScore:          10.0,
		WeightOfTheTextScore:           0.25,
		WeightOfTheAddressScore:        1.0,
		WeightOfThePhraseScore:         3.0,
		WeightOfTheCoordinationScore:   0.5,
		WeightOfTheNamedSiteEntryScore: 5,
		WeightOfTheOverlongTextScore:   5,
		WeightOfTheLinkSparsityPenalty: 1.5,
	}
}

func (weights ScoreWeights) relevanceOf(
	foundDocument queryanswers.FoundDocument,
	rarityOfTheQueryWords queryWordRarity,
	averageDocumentLength float64,
	queryWords []yacymodel.Hash,
) float64 {
	return weights.WeightOfTheTitleScore*titleScoreOf(
		foundDocument,
		rarityOfTheQueryWords,
		queryWords,
	) +
		weights.WeightOfTheTextScore*
			textScoreOf(
				foundDocument,
				rarityOfTheQueryWords,
				averageDocumentLength,
				queryWords,
			) +
		weights.WeightOfTheAddressScore*addressScoreOf(
			foundDocument,
			queryWords,
		) +
		weights.WeightOfThePhraseScore*phraseScoreOf(
			foundDocument,
		) +
		weights.WeightOfTheCoordinationScore*coordinationScoreOf(
			foundDocument,
			queryWords,
		) +
		weights.WeightOfTheNamedSiteEntryScore*namedSiteEntryScoreOf(
			foundDocument,
			queryWords,
		) -
		weights.WeightOfTheOverlongTextScore*overlongTextScoreOf(
			foundDocument,
		) -
		weights.WeightOfTheLinkSparsityPenalty*linkSparsityPenaltyOf(
			foundDocument,
		)
}
