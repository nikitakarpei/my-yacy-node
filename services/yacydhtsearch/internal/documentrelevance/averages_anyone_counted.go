package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

const (
	averageAmountOfWordsWhenNobodyCountedAWord       = 0.0
	averageLinkSparsityPenaltyWhenNobodyCountedALink = 0.0
)

type averagesAnyoneCounted struct {
	amountOfWords       float64
	linkSparsityPenalty float64
}

func averagesAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) averagesAnyoneCounted {
	return averagesAnyoneCounted{
		amountOfWords:       averageAmountOfWordsAnyoneCountedAmong(factsPerDocument),
		linkSparsityPenalty: averageLinkSparsityPenaltyAnyoneCountedAmong(factsPerDocument),
	}
}

func averageAmountOfWordsAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) float64 {
	sumOfTheAmountsOfWords, amountOfCountedDocuments := 0, 0
	for _, facts := range factsPerDocument {
		amountOfWords, counted := facts.AmountOfWords.Get()
		if !counted || amountOfWords <= 0 {
			continue
		}
		sumOfTheAmountsOfWords += amountOfWords
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return averageAmountOfWordsWhenNobodyCountedAWord
	}

	return float64(sumOfTheAmountsOfWords) / float64(amountOfCountedDocuments)
}

func averageLinkSparsityPenaltyAnyoneCountedAmong(
	factsPerDocument queryanswers.FactsPerDocument,
) float64 {
	sumOfThePenalties, amountOfCountedDocuments := 0.0, 0
	for _, facts := range factsPerDocument {
		penalty, counted := linkSparsityPenaltyOfTheCountedLinks(facts).Get()
		if !counted {
			continue
		}
		sumOfThePenalties += penalty
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return averageLinkSparsityPenaltyWhenNobodyCountedALink
	}

	return sumOfThePenalties / float64(amountOfCountedDocuments)
}
