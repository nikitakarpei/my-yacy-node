package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func peersAcrossQueryWords(
	chosenPeersPerQueryWord [][]peerchoice.ChosenPeer,
) []peerdirectory.AskablePeer {
	var chosenPeers []peerdirectory.AskablePeer
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeer := range chosenPeersOfQueryWord {
			if _, taken := takenPeers[chosenPeer.Peer.Hash]; taken {
				continue
			}
			takenPeers[chosenPeer.Peer.Hash] = struct{}{}
			chosenPeers = append(chosenPeers, chosenPeer.Peer)
		}
	}

	return chosenPeers
}
