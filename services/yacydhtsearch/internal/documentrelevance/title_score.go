package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfTheRarityOfTheQueryWordsADocumentWithoutATitleLoses = -1.0
	shareOfTheRarityOfTheQueryWordsOfATitleWithoutAQueryWord   = 0.0
)

func titleScoreOf(item peeranswers.AnsweredItem, rarity queryWordRarity) float64 {
	if item.Metadata.Title == "" {
		return shareOfTheRarityOfTheQueryWordsADocumentWithoutATitleLoses
	}
	if rarity.sumOfTheRarityOfTheQueryWords <= 0 {
		return shareOfTheRarityOfTheQueryWordsOfATitleWithoutAQueryWord
	}

	return rarity.sumOfTheRarityOf(queryWordsOfTheTitleOf(item)) /
		rarity.sumOfTheRarityOfTheQueryWords
}

func queryWordsOfTheTitleOf(item peeranswers.AnsweredItem) []yacymodel.Hash {
	wordsOfTheTitle := wordsOfTheTitleOf(item.Metadata.Title)

	queryWordsOfTheTitle := make([]yacymodel.Hash, 0, len(item.MatchedWords))
	for _, word := range matchedWordsAlwaysInTheSameOrder(item) {
		if _, inTheTitle := wordsOfTheTitle[word]; !inTheTitle {
			continue
		}
		queryWordsOfTheTitle = append(queryWordsOfTheTitle, word)
	}

	return queryWordsOfTheTitle
}

func wordsOfTheTitleOf(title string) map[yacymodel.Hash]struct{} {
	spelledWords := yacymodel.WordsIn(title)

	wordsOfTheTitle := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsOfTheTitle[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsOfTheTitle
}
