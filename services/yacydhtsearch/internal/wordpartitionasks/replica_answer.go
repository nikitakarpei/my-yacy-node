package wordpartitionasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAnswer struct {
	Holder                peerdirectory.AskablePeer
	ListedDocuments       yacymodel.URLHashes
	MatchedDocuments      []MatchedDocument
	AmountOfDocumentsHeld yacymodel.Optional[int]
	Searched              bool
}

type MatchedDocument struct {
	Metadata yacymodel.URLMetadata
	Posting  yacymodel.Optional[yacymodel.RWIPosting]
}
