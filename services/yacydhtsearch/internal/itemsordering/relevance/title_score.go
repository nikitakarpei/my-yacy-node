package relevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const titleScoreOfADocumentWithoutATitle = -2.0

func titleScoreOf(item peeranswers.AnsweredItem) float64 {
	if item.Metadata.Title == "" {
		return titleScoreOfADocumentWithoutATitle
	}
	wordsOfTheTitle := wordsOfTheTitleOf(item.Metadata.Title)

	amountOfQueryWordsInTheTitle := 0
	for word := range item.MatchedWords {
		if _, inTheTitle := wordsOfTheTitle[word]; !inTheTitle {
			continue
		}
		amountOfQueryWordsInTheTitle++
	}

	return float64(amountOfQueryWordsInTheTitle)
}

func wordsOfTheTitleOf(title string) map[yacymodel.Hash]struct{} {
	spelledWords := yacymodel.WordsIn(title)

	wordsOfTheTitle := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsOfTheTitle[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsOfTheTitle
}
