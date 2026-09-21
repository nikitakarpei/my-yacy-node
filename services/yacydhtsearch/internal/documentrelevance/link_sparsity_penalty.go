package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	leastLinkDensityWithoutASparsityPenalty = 0.03
	sparsityPenaltyWhenNobodyCountedALink   = 0.0
)

func linkSparsityPenaltyOf(
	facts queryanswers.DocumentFacts,
	averageLinkSparsityPenaltyAnyoneCounted float64,
) float64 {
	return sparsityPenaltyOfTheCountedLinks(facts).
		OrElse(averageLinkSparsityPenaltyAnyoneCounted)
}

func averageLinkSparsityPenaltyAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) float64 {
	sumOfThePenalties, amountOfCountedDocuments := 0.0, 0
	for _, facts := range factsPerDocument {
		penalty, counted := sparsityPenaltyOfTheCountedLinks(facts).Get()
		if !counted {
			continue
		}
		sumOfThePenalties += penalty
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return sparsityPenaltyWhenNobodyCountedALink
	}

	return sumOfThePenalties / float64(amountOfCountedDocuments)
}

func sparsityPenaltyOfTheCountedLinks(
	facts queryanswers.DocumentFacts,
) yacymodel.Optional[float64] {
	amountOfLinks, countedForTheDocument := facts.AmountOfLinks.Get()
	amountOfWords, counted := facts.AmountOfWords.Get()
	if !countedForTheDocument || !counted || amountOfWords <= 0 {
		return yacymodel.None[float64]()
	}
	linkDensity := linkDensityOf(amountOfLinks, amountOfWords)
	if linkDensity >= leastLinkDensityWithoutASparsityPenalty {
		return yacymodel.Some(0.0)
	}

	return yacymodel.Some(1 - linkDensity/leastLinkDensityWithoutASparsityPenalty)
}

func linkDensityOf(amountOfLinks int, amountOfWords int) float64 {
	return float64(amountOfLinks) / float64(amountOfWords)
}
