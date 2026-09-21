package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const leastLinkDensityWithoutSparsityPenalty = 0.03

type linkSparsityPenalty struct {
	averageLinkSparsityPenalty float64
}

func linkSparsityPenaltyFrom(statistics answersStatistics) linkSparsityPenalty {
	return linkSparsityPenalty{
		averageLinkSparsityPenalty: statistics.documentAverages.averageLinkSparsityPenalty,
	}
}

func (penalty linkSparsityPenalty) penaltyOf(document queryanswers.FoundDocument) float64 {
	return countedLinkSparsityPenaltyOf(document).OrElse(penalty.averageLinkSparsityPenalty)
}

func countedLinkSparsityPenaltyOf(
	document queryanswers.FoundDocument,
) yacymodel.Optional[float64] {
	amountOfLinks, linksCounted := document.Facts.AmountOfLinks.Get()
	amountOfWords, wordsCounted := document.Facts.AmountOfWords.Get()
	if !linksCounted || !wordsCounted || amountOfWords <= 0 {
		return yacymodel.None[float64]()
	}
	linkDensity := float64(amountOfLinks) / float64(amountOfWords)
	if linkDensity >= leastLinkDensityWithoutSparsityPenalty {
		return yacymodel.Some(0.0)
	}

	return yacymodel.Some(1 - linkDensity/leastLinkDensityWithoutSparsityPenalty)
}
