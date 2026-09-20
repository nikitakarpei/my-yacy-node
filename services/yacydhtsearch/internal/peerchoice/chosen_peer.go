package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type ChosenPeer struct {
	Peer        peerdirectory.AskablePeer
	Partition   uint
	Reliability float64
}
