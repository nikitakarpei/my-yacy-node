package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ChosenPeer struct {
	Peer      peerdirectory.AskablePeer
	Partition uint
}

type PeersOfQueryWord struct {
	Partitions  yacymodel.DHTRingPartitions
	ChosenPeers []ChosenPeer
}

func (p PeersOfQueryWord) Peers() []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(p.ChosenPeers))
	for _, chosenPeer := range p.ChosenPeers {
		peers = append(peers, chosenPeer.Peer)
	}

	return peers
}

func (p PeersOfQueryWord) PeersPerPartition() [][]peerdirectory.AskablePeer {
	peersPerPartition := make([][]peerdirectory.AskablePeer, p.Partitions)
	for _, chosenPeer := range p.ChosenPeers {
		peersPerPartition[chosenPeer.Partition] = append(
			peersPerPartition[chosenPeer.Partition], chosenPeer.Peer,
		)
	}

	return peersPerPartition
}
