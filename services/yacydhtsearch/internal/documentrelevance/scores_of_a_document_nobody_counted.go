package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const scoreOfADocumentWhenNoDocumentWasCounted = 0.0

type scoresOfADocumentNobodyCounted struct {
	phraseScore         float64
	linkSparsityPenalty float64
}

func scoresOfADocumentNobodyCountedAmong(
	foundDocuments []queryanswers.FoundDocument,
) scoresOfADocumentNobodyCounted {
	return scoresOfADocumentNobodyCounted{
		phraseScore: averageScoreOfTheCountedDocumentsAmong(foundDocuments, phraseScoreOf),
		linkSparsityPenalty: averageScoreOfTheCountedDocumentsAmong(
			foundDocuments, linkSparsityPenaltyOf,
		),
	}
}

func averageScoreOfTheCountedDocumentsAmong(
	foundDocuments []queryanswers.FoundDocument,
	scoreOfTheDocument func(queryanswers.FoundDocument) yacymodel.Optional[float64],
) float64 {
	sumOfTheScores, amountOfCountedDocuments := 0.0, 0
	for _, foundDocument := range foundDocuments {
		score, counted := scoreOfTheDocument(foundDocument).Get()
		if !counted {
			continue
		}
		sumOfTheScores += score
		amountOfCountedDocuments++
	}
	if amountOfCountedDocuments == 0 {
		return scoreOfADocumentWhenNoDocumentWasCounted
	}

	return sumOfTheScores / float64(amountOfCountedDocuments)
}
