package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const titleScoreOfDocumentWithoutTitle = -1.0

type titleScorer struct {
	queryWordRarities queryWordRarities
	queryWords        []yacymodel.Hash
}

func titleScorerFrom(statistics answersStatistics) titleScorer {
	return titleScorer{
		queryWordRarities: statistics.queryWordRarities,
		queryWords:        statistics.queryWords,
	}
}

func (scorer titleScorer) scoreOf(document queryanswers.FoundDocument) float64 {
	if document.Title == "" {
		return titleScoreOfDocumentWithoutTitle
	}

	return scorer.queryWordRarities.rarityShareOfWords(scorer.queryWordsInTitleOf(document))
}

func (scorer titleScorer) queryWordsInTitleOf(
	document queryanswers.FoundDocument,
) []yacymodel.Hash {
	wordsInTitle := wordsIn(document.Title)

	queryWordsInTitle := make([]yacymodel.Hash, 0, len(scorer.queryWords))
	for _, word := range scorer.queryWords {
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
