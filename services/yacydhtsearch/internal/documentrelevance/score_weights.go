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
	facts queryanswers.DocumentFacts,
	rarityOfTheQuery queryRarity,
	averageFacts averageFactsAnyoneCounted,
	queryWords []yacymodel.Hash,
) float64 {
	return weights.WeightOfTheTitleScore*titleScoreOf(
		foundDocument,
		rarityOfTheQuery,
		queryWords,
	) +
		weights.WeightOfTheTextScore*
			textScoreOf(
				facts,
				rarityOfTheQuery,
				averageFacts.amountOfWords,
				queryWords,
			) +
		weights.WeightOfThePhraseScore*phraseScoreOf(facts) +
		weights.WeightOfTheNamedSiteEntryScore*namedSiteEntryScoreOf(
			foundDocument,
			queryWords,
		) -
		weights.WeightOfTheLinkSparsityPenalty*linkSparsityPenaltyOf(
			facts, averageFacts.linkSparsityPenalty,
		)
}
