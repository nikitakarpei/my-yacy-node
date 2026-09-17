package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ChosenPeersPerQueryWord []ChosenPeersOfQueryWord

func (chosenPeersPerQueryWord ChosenPeersPerQueryWord) PeersAcrossQueryWords() []peerdirectory.AskablePeer {
	var peersAcrossQueryWords []peerdirectory.AskablePeer
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
			if _, taken := takenPeers[chosenPeer.Peer.Hash]; taken {
				continue
			}
			takenPeers[chosenPeer.Peer.Hash] = struct{}{}
			peersAcrossQueryWords = append(peersAcrossQueryWords, chosenPeer.Peer)
		}
	}

	return peersAcrossQueryWords
}
