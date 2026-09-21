package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

const (
	averageAmountOfWordsWhenNoPageRead       = 0.0
	averageLinkSparsityPenaltyWhenNoPageRead = 0.0
)

type documentAverages struct {
	averageAmountOfWords       float64
	averageLinkSparsityPenalty float64
}

func documentAveragesAmong(
	documents []queryanswers.FoundDocument,
) documentAverages {
	return documentAverages{
		averageAmountOfWords:       averageAmountOfWordsAmong(documents),
		averageLinkSparsityPenalty: averageLinkSparsityPenaltyAmong(documents),
	}
}

func averageAmountOfWordsAmong(
	documents []queryanswers.FoundDocument,
) float64 {
	sumOfAmountsOfWords, amountOfCountedDocuments := 0, 0
	for _, document := range documents {
		amountOfWords, wordsCounted := document.Facts.AmountOfWords.Get()
		if !wordsCounted || amountOfWords <= 0 {
			continue
		}
		sumOfAmountsOfWords += amountOfWords
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return averageAmountOfWordsWhenNoPageRead
	}

	return float64(sumOfAmountsOfWords) / float64(amountOfCountedDocuments)
}

func averageLinkSparsityPenaltyAmong(
	documents []queryanswers.FoundDocument,
) float64 {
	sumOfPenalties, amountOfCountedDocuments := 0.0, 0
	for _, document := range documents {
		penalty, penaltyCounted := countedLinkSparsityPenaltyOf(document).Get()
		if !penaltyCounted {
			continue
		}
		sumOfPenalties += penalty
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return averageLinkSparsityPenaltyWhenNoPageRead
	}

	return sumOfPenalties / float64(amountOfCountedDocuments)
}
