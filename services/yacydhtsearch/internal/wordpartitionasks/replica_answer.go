package wordpartitionasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// TECHDEBT: Vocabulary — the peer that sent a document is Replica here and Holder in queryfindings.MetadataReplica.
type ReplicaAnswer struct {
	Replica               peerdirectory.AskablePeer
	ListedDocuments       []ListedDocument
	AmountOfDocumentsHeld yacymodel.Optional[int]
	Searched              bool
}
