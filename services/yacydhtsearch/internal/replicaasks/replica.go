package replicaasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Replica struct {
	Peer      peerdirectory.AskablePeer
	Word      yacymodel.Hash
	Partition uint
}
