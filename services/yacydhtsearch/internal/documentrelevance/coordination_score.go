package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const coordinationScoreOfAQueryOfOneWord = 1.0

func coordinationScoreOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) float64 {
	if len(queryWords) <= 1 {
		return coordinationScoreOfAQueryOfOneWord
	}

	amountOfQueryWordsTheDocumentHolds := 0
	for _, word := range queryWords {
		if foundDocument.HitsPerQueryWord[word] <= 0 {
			continue
		}
		amountOfQueryWordsTheDocumentHolds++
	}

	return float64(amountOfQueryWordsTheDocumentHolds) / float64(len(queryWords))
}
