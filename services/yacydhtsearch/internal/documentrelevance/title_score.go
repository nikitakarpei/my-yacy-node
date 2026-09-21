package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const titleScoreOfDocumentWithoutTitle = -1.0

func titleScoreOf(
	foundDocument queryanswers.FoundDocument,
	queryWordRarities queryWordRarities,
	queryWords []yacymodel.Hash,
) float64 {
	if foundDocument.Title == "" {
		return titleScoreOfDocumentWithoutTitle
	}

	return queryWordRarities.rarityShareOfWords(queryWordsInTitleOf(foundDocument, queryWords))
}

func queryWordsInTitleOf(
	foundDocument queryanswers.FoundDocument, queryWords []yacymodel.Hash,
) []yacymodel.Hash {
	wordsInTitle := wordsIn(foundDocument.Title)

	queryWordsInTitle := make([]yacymodel.Hash, 0, len(queryWords))
	for _, word := range queryWords {
		if _, inTitle := wordsInTitle[word]; !inTitle {
			continue
		}
		queryWordsInTitle = append(queryWordsInTitle, word)
	}

	return queryWordsInTitle
}

func wordsIn(text string) map[yacymodel.Hash]struct{} {
	spelledWords := yacymodel.WordsIn(text)

	words := make(map[yacymodel.Hash]struct{}, len(spelledWords))
	for _, spelledWord := range spelledWords {
		words[yacymodel.WordHash(spelledWord)] = struct{}{}
	}

	return words
}
