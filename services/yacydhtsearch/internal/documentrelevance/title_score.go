package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfTheQueryRarityADocumentWithoutATitleLoses = -1.0

func titleScoreOf(
	foundDocument queryanswers.FoundDocument,
	rarityOfTheQuery queryRarity,
	queryWords []yacymodel.Hash,
) float64 {
	if foundDocument.Title == "" {
		return shareOfTheQueryRarityADocumentWithoutATitleLoses
	}

	return rarityOfTheQuery.shareHeldBy(queryWordsInTheTitleOf(foundDocument, queryWords))
}

func queryWordsInTheTitleOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) []yacymodel.Hash {
	wordsInTheTitle := wordsInTheTitleOf(foundDocument)

	queryWordsInTheTitle := make([]yacymodel.Hash, 0, len(queryWords))
	for _, word := range queryWords {
		if _, inTheTitle := wordsInTheTitle[word]; !inTheTitle {
			continue
		}
		queryWordsInTheTitle = append(queryWordsInTheTitle, word)
	}

	return queryWordsInTheTitle
}

func wordsInTheTitleOf(
	foundDocument queryanswers.FoundDocument,
) map[yacymodel.Hash]struct{} {
	spelledWords := yacymodel.WordsIn(foundDocument.Title)

	wordsInTheTitle := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		wordsInTheTitle[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return wordsInTheTitle
}
