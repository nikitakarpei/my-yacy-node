package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const leastLinkDensityWithoutASparsityPenalty = 0.03

func linkSparsityPenaltyOf(foundDocument queryanswers.FoundDocument) float64 {
	linkCounts, sentForTheDocument := foundDocument.LinkCounts.Get()
	if !sentForTheDocument || foundDocument.AmountOfWords <= 0 {
		return 0
	}
	linkDensity := linkDensityOf(linkCounts.AmountOfLinks(), foundDocument.AmountOfWords)
	if linkDensity >= leastLinkDensityWithoutASparsityPenalty {
		return 0
	}

	return 1 - linkDensity/leastLinkDensityWithoutASparsityPenalty
}

func linkDensityOf(amountOfLinks int, amountOfWords int) float64 {
	return float64(amountOfLinks) / float64(amountOfWords)
}
