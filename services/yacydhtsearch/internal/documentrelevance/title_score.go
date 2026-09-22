package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

const (
	titleScoreOfDocumentWithoutTitle    = -1.0
	titleScoreFactorOfWholeQueryInTitle = 1.5
)

type titleScorer struct {
	queryWordRarities queryWordRarities
	queryVocabulary   queryVocabulary
}

func titleScorerFrom(statistics answersStatistics) titleScorer {
	return titleScorer{
		queryWordRarities: statistics.queryWordRarities,
		queryVocabulary:   statistics.queryVocabulary,
	}
}

func (scorer titleScorer) scoreOf(document queryanswers.FoundDocument) float64 {
	if document.Title == "" {
		return titleScoreOfDocumentWithoutTitle
	}
	queryWordsInTitle := scorer.queryVocabulary.wordsIn(document.Title)
	titleScore := scorer.queryWordRarities.rarityShareOfWords(queryWordsInTitle)
	if len(queryWordsInTitle) == len(scorer.queryVocabulary.words) {
		return titleScore * titleScoreFactorOfWholeQueryInTitle
	}

	return titleScore
}
