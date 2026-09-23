package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ChosenPeersPerQueryWord []ChosenPeersOfQueryWord

func (chosenPeersPerQueryWord ChosenPeersPerQueryWord) PeersAcrossQueryWords() []peerdirectory.AskablePeer {
	chosenPeersAcrossQueryWords := chosenPeersPerQueryWord.ChosenPeersAcrossQueryWords()

	peersAcrossQueryWords := make([]peerdirectory.AskablePeer, 0, len(chosenPeersAcrossQueryWords))
	for _, chosenPeer := range chosenPeersAcrossQueryWords {
		peersAcrossQueryWords = append(peersAcrossQueryWords, chosenPeer.Peer)
	}

	return peersAcrossQueryWords
}

func (chosenPeersPerQueryWord ChosenPeersPerQueryWord) ChosenPeersAcrossQueryWords() []ChosenPeer {
	var chosenPeersAcrossQueryWords []ChosenPeer
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
			if _, taken := takenPeers[chosenPeer.Peer.Hash]; taken {
				continue
			}
			takenPeers[chosenPeer.Peer.Hash] = struct{}{}
			chosenPeersAcrossQueryWords = append(chosenPeersAcrossQueryWords, chosenPeer)
		}
	}

	return chosenPeersAcrossQueryWords
}
