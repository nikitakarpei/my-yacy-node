package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

type answersStatistics struct {
	queryVocabulary   queryVocabulary
	queryWordRarities queryWordRarities
	documentAverages  documentAverages
}

func answersStatisticsFrom(answers queryanswers.AnsweredQuery) answersStatistics {
	return answersStatistics{
		queryVocabulary:   queryVocabularyOf(answers),
		queryWordRarities: queryWordRaritiesFrom(answers),
		documentAverages:  documentAveragesAmong(answers.FoundDocuments),
	}
}
