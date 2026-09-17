package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type crossCheckedDocumentsRound struct {
	asks                                                 []peerasks.CrossCheckedDocumentsAsk
	answeredAsks                                         []peerasks.AnsweredCrossCheckedDocumentsAsk
	amountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
}

func (round crossCheckedDocumentsRound) documentsFoundByCrossCheckingPerQueryWord() documentsPerQueryWord {
	documentsFoundByCrossCheckingPerQueryWord := documentsPerQueryWord{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] = distinctDocuments{}
		}
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word].add(document)
		}
	}

	return documentsFoundByCrossCheckingPerQueryWord
}
