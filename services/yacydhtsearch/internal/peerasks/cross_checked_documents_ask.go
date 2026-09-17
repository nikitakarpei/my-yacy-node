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
	Ask                     CrossCheckedDocumentsAsk
	DocumentsHeldForTheWord []yacymodel.URLHash
}
