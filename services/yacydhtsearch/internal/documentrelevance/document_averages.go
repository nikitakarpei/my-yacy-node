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
	foundDocuments []queryanswers.FoundDocument,
) documentAverages {
	return documentAverages{
		averageAmountOfWords:       averageAmountOfWordsAmong(foundDocuments),
		averageLinkSparsityPenalty: averageLinkSparsityPenaltyAmong(foundDocuments),
	}
}

func averageAmountOfWordsAmong(
	foundDocuments []queryanswers.FoundDocument,
) float64 {
	sumOfAmountsOfWords, amountOfCountedDocuments := 0, 0
	for _, foundDocument := range foundDocuments {
		amountOfWords, wordsCounted := foundDocument.Facts.AmountOfWords.Get()
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
	foundDocuments []queryanswers.FoundDocument,
) float64 {
	sumOfPenalties, amountOfCountedDocuments := 0.0, 0
	for _, foundDocument := range foundDocuments {
		penalty, penaltyCounted := countedLinkSparsityPenaltyOf(foundDocument).Get()
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
