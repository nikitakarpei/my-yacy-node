package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const coordinationScoreOfAQueryOfOneWord = 1.0

func coordinationScoreOf(
	item peeranswers.AnsweredItem, queryWords []yacymodel.Hash,
) float64 {
	if len(queryWords) <= 1 {
		return coordinationScoreOfAQueryOfOneWord
	}

	amountOfQueryWordsTheDocumentHolds := 0
	for _, word := range queryWords {
		if item.MatchedWords[word].Hits <= 0 {
			continue
		}
		amountOfQueryWordsTheDocumentHolds++
	}

	return float64(amountOfQueryWordsTheDocumentHolds) / float64(len(queryWords))
}
