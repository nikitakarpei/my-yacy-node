package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ScoreWeights struct {
	WeightOfTheTitleScore          float64
	WeightOfTheTextScore           float64
	WeightOfThePhraseScore         float64
	WeightOfTheNamedSiteEntryScore float64
	WeightOfTheLinkSparsityPenalty float64
}

func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		WeightOfTheTitleScore:          10.0,
		WeightOfTheTextScore:           0.25,
		WeightOfThePhraseScore:         3.0,
		WeightOfTheNamedSiteEntryScore: 5,
		WeightOfTheLinkSparsityPenalty: 1.5,
	}
}

func (weights ScoreWeights) relevanceOf(
	foundDocument queryanswers.FoundDocument,
	rarityOfTheQueryWords queryWordRarity,
	averageLengthOfTheReadPages float64,
	scoresOfADocumentNobodyCounted scoresOfADocumentNobodyCounted,
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
				averageLengthOfTheReadPages,
				queryWords,
			) +
		weights.WeightOfThePhraseScore*phraseScoreOf(foundDocument).
			OrElse(scoresOfADocumentNobodyCounted.phraseScore) +
		weights.WeightOfTheNamedSiteEntryScore*namedSiteEntryScoreOf(
			foundDocument,
			queryWords,
		) -
		weights.WeightOfTheLinkSparsityPenalty*linkSparsityPenaltyOf(foundDocument).
			OrElse(scoresOfADocumentNobodyCounted.linkSparsityPenalty)
}
