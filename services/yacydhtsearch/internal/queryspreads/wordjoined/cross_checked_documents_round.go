package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"

type crossCheckedDocumentsRound struct {
	candidates   crossCheckCandidates
	asks         []peerasks.MatchedAndHeldDocumentsAsk
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk
}

func (round crossCheckedDocumentsRound) documentsFoundByCrossCheckingPerQueryWord() documentsPerQueryWord {
	documentsFoundByCrossCheckingPerQueryWord := documentsPerQueryWord{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] = distinctDocuments{}
		}
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word].add(document)
		}
	}

	return documentsFoundByCrossCheckingPerQueryWord
}
