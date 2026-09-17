package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type HeldDocumentsAsk struct {
	Peer      peerdirectory.AskablePeer
	Word      yacymodel.Hash
	Documents []yacymodel.URLHash
}

type AnsweredHeldDocumentsAsk struct {
	Ask                     HeldDocumentsAsk
	DocumentsHeldForTheWord []yacymodel.URLHash
}
