package wordpartitionasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAnswer struct {
	Replica               peerdirectory.AskablePeer
	ListedDocuments       []ListedDocument
	AmountOfDocumentsHeld yacymodel.Optional[int]
	Searched              bool
}
