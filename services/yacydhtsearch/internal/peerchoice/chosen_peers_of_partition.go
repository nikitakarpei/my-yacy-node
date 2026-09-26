package peerchoice

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"

type ChosenPeersOfPartition struct {
	Partition uint
	Peers     []peerdirectory.AskablePeer
}
