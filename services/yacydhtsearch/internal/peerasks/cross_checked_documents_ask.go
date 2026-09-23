package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type CrossCheckedDocumentsAsk struct {
	Peer      peerdirectory.AskablePeer
	Word      yacymodel.Hash
	Documents []yacymodel.URLHash
}

type AnsweredCrossCheckedDocumentsAsk struct {
	Ask         CrossCheckedDocumentsAsk
	Abstract    []yacymodel.URLHash
	PeerVersion string
}

func (answeredAsk AnsweredCrossCheckedDocumentsAsk) Answers(ask CrossCheckedDocumentsAsk) bool {
	return answeredAsk.Ask.Peer.Hash == ask.Peer.Hash && answeredAsk.Ask.Word == ask.Word
}
