package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfTheRarityOfTheQueryWordsADocumentWithoutATitleLoses = -1.0

func titleScoreOf(
	foundDocument queryanswers.FoundDocument, rarity queryWordRarity, queryWords []yacymodel.Hash,
) float64 {
	if foundDocument.Title == "" {
		return shareOfTheRarityOfTheQueryWordsADocumentWithoutATitleLoses
	}

	return rarity.sumOfTheRarityOf(queryWordsOfTheTitleOf(foundDocument, queryWords)) /
		rarity.sumOfTheRarityOfTheQueryWords
}

func queryWordsOfTheTitleOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) []yacymodel.Hash {
	wordsOfTheTitle := wordsOfTheTitleOf(foundDocument.Title)

	queryWordsOfTheTitle := make([]yacymodel.Hash, 0, len(queryWords))
	for _, word := range queryWords {
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
