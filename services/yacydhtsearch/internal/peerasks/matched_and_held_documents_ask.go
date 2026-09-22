// Package peerasks holds the asks the service puts to peers and what one peer
// answered for each.
package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedAndHeldDocumentsAsk struct {
	Peer          peerdirectory.AskablePeer
	Partition     uint
	Word          yacymodel.Hash
	ExcludedWords []yacymodel.Hash
	Language      string
	ItemsCeiling  int
}

type AnsweredMatchedAndHeldDocumentsAsk struct {
	Ask                             MatchedAndHeldDocumentsAsk
	DocumentsListedForTheWord       []yacymodel.URLHash
	MatchedDocuments                []MatchedDocument
	AmountOfDocumentsHeldForTheWord yacymodel.Optional[int]
	PeerVersion                     string
}

func (answeredAsk AnsweredMatchedAndHeldDocumentsAsk) AnswersTheAskTo(
	peer peerdirectory.AskablePeer,
	word yacymodel.Hash,
) bool {
	return answeredAsk.Ask.Peer.Hash == peer.Hash && answeredAsk.Ask.Word == word
}
