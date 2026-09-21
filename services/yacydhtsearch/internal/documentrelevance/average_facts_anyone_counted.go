package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

type averageFactsAnyoneCounted struct {
	amountOfWords       float64
	linkSparsityPenalty float64
}

func averageFactsAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) averageFactsAnyoneCounted {
	return averageFactsAnyoneCounted{
		amountOfWords:       averageAmountOfWordsAnyoneCountedAmong(factsPerDocument),
		linkSparsityPenalty: averageLinkSparsityPenaltyAnyoneCountedAmong(factsPerDocument),
	}
}
