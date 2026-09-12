package relevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const coordinationScoreOfAQueryOfOneWord = 1.0

func coordinationScoreOf(item peeranswers.AnsweredItem, amountOfQueryWords int) float64 {
	if amountOfQueryWords <= 1 {
		return coordinationScoreOfAQueryOfOneWord
	}

	amountOfQueryWordsTheDocumentHolds := 0
	for _, count := range item.MatchedWords {
		if count.Hits <= 0 {
			continue
		}
		amountOfQueryWordsTheDocumentHolds++
	}

	return float64(amountOfQueryWordsTheDocumentHolds) / float64(amountOfQueryWords)
}

func amountOfQueryWordsAcross(items []peeranswers.AnsweredItem) int {
	queryWordsOfTheAnswers := map[yacymodel.Hash]struct{}{}
	for _, item := range items {
		for word := range item.MatchedWords {
			queryWordsOfTheAnswers[word] = struct{}{}
		}
	}

	return len(queryWordsOfTheAnswers)
}
