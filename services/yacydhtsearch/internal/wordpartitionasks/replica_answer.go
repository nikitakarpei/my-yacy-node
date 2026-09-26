package wordpartitionasks

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"

type ReplicaAnswer struct {
	Replica         peerdirectory.AskablePeer
	ListedDocuments []ListedDocument
	Searched        bool
}
