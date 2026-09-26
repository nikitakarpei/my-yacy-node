package peerchoice

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ChosenPeersOfQueryWord struct {
	QueryWord   yacymodel.Hash
	ChosenPeers []ChosenPeer
}

func (chosenPeers ChosenPeersOfQueryWord) ChosenPeersPerPartition() []ChosenPeersOfPartition {
	var chosenPeersPerPartition []ChosenPeersOfPartition
	placeOfEachPartition := map[uint]int{}
	for _, chosenPeer := range chosenPeers.ChosenPeers {
		place, placed := placeOfEachPartition[chosenPeer.Partition]
		if !placed {
			place = len(chosenPeersPerPartition)
			placeOfEachPartition[chosenPeer.Partition] = place
			chosenPeersPerPartition = append(
				chosenPeersPerPartition,
				ChosenPeersOfPartition{Partition: chosenPeer.Partition},
			)
		}
		chosenPeersPerPartition[place].Peers = append(
			chosenPeersPerPartition[place].Peers,
			chosenPeer.Peer,
		)
	}

	return chosenPeersPerPartition
}
