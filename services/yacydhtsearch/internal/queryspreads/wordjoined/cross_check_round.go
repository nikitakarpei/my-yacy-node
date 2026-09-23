package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"

type crossCheckRound struct {
	candidates   crossCheckCandidates
	asks         []peerasks.SearchDocumentsAsk
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk
}

func (round crossCheckRound) documentsPerQueryWord() documentsPerQueryWord {
	documentsOfEachQueryWord := documentsPerQueryWord{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsOfEachQueryWord[answeredAsk.Ask.Word] == nil {
			documentsOfEachQueryWord[answeredAsk.Ask.Word] = distinctDocuments{}
		}
		for _, document := range answeredAsk.Abstract {
			documentsOfEachQueryWord[answeredAsk.Ask.Word].add(document)
		}
	}

	return documentsOfEachQueryWord
}
