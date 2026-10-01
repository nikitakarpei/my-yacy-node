package documentrelevance

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"

type findingsStatistics struct {
	queryVocabulary   queryVocabulary
	queryWordRarities queryWordRarities
	documentAverages  documentAverages
}

func findingsStatisticsFrom(findings queryfindings.Findings) findingsStatistics {
	return findingsStatistics{
		queryVocabulary:   queryVocabularyOf(findings),
		queryWordRarities: queryWordRaritiesFrom(findings),
		documentAverages:  documentAveragesAmong(findings.FoundDocuments),
	}
}
