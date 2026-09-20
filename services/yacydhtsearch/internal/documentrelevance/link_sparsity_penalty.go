package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const leastLinkDensityWithoutASparsityPenalty = 0.03

func linkSparsityPenaltyOf(foundDocument queryanswers.FoundDocument) yacymodel.Optional[float64] {
	linkCounts, countedForTheDocument := foundDocument.LinkCounts.Get()
	amountOfWords, counted := foundDocument.AmountOfWordsAnyoneCounted().Get()
	if !countedForTheDocument || !counted || amountOfWords <= 0 {
		return yacymodel.None[float64]()
	}
	linkDensity := linkDensityOf(linkCounts.AmountOfLinks(), amountOfWords)
	if linkDensity >= leastLinkDensityWithoutASparsityPenalty {
		return yacymodel.Some(0.0)
	}

	return yacymodel.Some(1 - linkDensity/leastLinkDensityWithoutASparsityPenalty)
}

func linkDensityOf(amountOfLinks int, amountOfWords int) float64 {
	return float64(amountOfLinks) / float64(amountOfWords)
}
