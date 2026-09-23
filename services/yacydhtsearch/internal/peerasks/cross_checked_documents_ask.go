package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type CrossCheckedDocumentsAsk struct {
	Peer         peerdirectory.AskablePeer
	Partition    uint
	Word         yacymodel.Hash
	Documents    []yacymodel.URLHash
	ItemsCeiling int
}

type AnsweredCrossCheckedDocumentsAsk struct {
	Ask                             CrossCheckedDocumentsAsk
	DocumentsListedForTheWord       []yacymodel.URLHash
	AmountOfDocumentsHeldForTheWord yacymodel.Optional[int]
	PeerVersion                     string
}

func (answeredAsk AnsweredCrossCheckedDocumentsAsk) Answers(ask CrossCheckedDocumentsAsk) bool {
	return answeredAsk.Ask.Peer.Hash == ask.Peer.Hash && answeredAsk.Ask.Word == ask.Word
}
