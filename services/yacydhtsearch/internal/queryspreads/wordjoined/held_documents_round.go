package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type heldDocumentsRound struct {
	asks                                         []peerasks.HeldDocumentsAsk
	answeredAsks                                 []peerasks.AnsweredHeldDocumentsAsk
	amountOfDocumentsPastTheHeldDocumentsCeiling int
}

func (round heldDocumentsRound) documentsHeldPerQueryWord() map[yacymodel.Hash]map[yacymodel.URLHash]struct{} {
	documentsHeldPerQueryWord := map[yacymodel.Hash]map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsHeldPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsHeldPerQueryWord[answeredAsk.Ask.Word] = map[yacymodel.URLHash]struct{}{}
		}
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			documentsHeldPerQueryWord[answeredAsk.Ask.Word][document] = struct{}{}
		}
	}

	return documentsHeldPerQueryWord
}
