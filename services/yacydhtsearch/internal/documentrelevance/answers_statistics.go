package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type answersStatistics struct {
	queryWords        []yacymodel.Hash
	queryWordRarities queryWordRarities
	documentAverages  documentAverages
}

func answersStatisticsFrom(answers queryanswers.AnsweredQuery) answersStatistics {
	return answersStatistics{
		queryWords:        answers.QueryWords,
		queryWordRarities: queryWordRaritiesFrom(answers),
		documentAverages:  documentAveragesAmong(answers.FoundDocuments),
	}
}
