package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type crossCheckedDocumentsRound struct {
	asks                                                 []peerasks.CrossCheckedDocumentsAsk
	answeredAsks                                         []peerasks.AnsweredCrossCheckedDocumentsAsk
	amountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
}

func (round crossCheckedDocumentsRound) documentsFoundByCrossCheckingPerQueryWord() map[yacymodel.Hash]map[yacymodel.URLHash]struct{} {
	documentsFoundByCrossCheckingPerQueryWord := map[yacymodel.Hash]map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] = map[yacymodel.URLHash]struct{}{}
		}
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word][document] = struct{}{}
		}
	}

	return documentsFoundByCrossCheckingPerQueryWord
}
