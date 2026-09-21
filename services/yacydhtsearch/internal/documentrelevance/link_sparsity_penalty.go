package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const leastLinkDensityWithoutSparsityPenalty = 0.03

func linkSparsityPenaltyOf(
	foundDocument queryanswers.FoundDocument,
	averageLinkSparsityPenalty float64,
) float64 {
	return countedLinkSparsityPenaltyOf(foundDocument).
		OrElse(averageLinkSparsityPenalty)
}

func countedLinkSparsityPenaltyOf(
	foundDocument queryanswers.FoundDocument,
) yacymodel.Optional[float64] {
	amountOfLinks, linksCounted := foundDocument.Facts.AmountOfLinks.Get()
	amountOfWords, wordsCounted := foundDocument.Facts.AmountOfWords.Get()
	if !linksCounted || !wordsCounted || amountOfWords <= 0 {
		return yacymodel.None[float64]()
	}
	linkDensity := linkDensityOf(amountOfLinks, amountOfWords)
	if linkDensity >= leastLinkDensityWithoutSparsityPenalty {
		return yacymodel.Some(0.0)
	}

	return yacymodel.Some(1 - linkDensity/leastLinkDensityWithoutSparsityPenalty)
}

func linkDensityOf(amountOfLinks int, amountOfWords int) float64 {
	return float64(amountOfLinks) / float64(amountOfWords)
}
