package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"

type crossCheckRound struct {
	candidates   crossCheckCandidates
	asks         []peerasks.SearchDocumentsAsk
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk
}

func (round crossCheckRound) documentsFoundByCrossCheckingPerQueryWord() documentsPerQueryWord {
	documentsFoundByCrossCheckingPerQueryWord := documentsPerQueryWord{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] = distinctDocuments{}
		}
		for _, document := range answeredAsk.Abstract {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word].add(document)
		}
	}

	return documentsFoundByCrossCheckingPerQueryWord
}
