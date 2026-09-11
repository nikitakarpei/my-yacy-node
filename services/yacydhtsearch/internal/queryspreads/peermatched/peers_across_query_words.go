package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func peersAcrossQueryWords(
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
	peerCallsCeiling int,
) []peerdirectory.AskablePeer {
	chosenPeers := make([]peerdirectory.AskablePeer, 0, peerCallsCeiling)
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, peersOfQueryWord := range chosenPeersPerQueryWord {
		for _, peer := range peersOfQueryWord {
			if _, taken := takenPeers[peer.Hash]; taken {
				continue
			}
			if len(chosenPeers) == peerCallsCeiling {
				return chosenPeers
			}
			takenPeers[peer.Hash] = struct{}{}
			chosenPeers = append(chosenPeers, peer)
		}
	}

	return chosenPeers
}
