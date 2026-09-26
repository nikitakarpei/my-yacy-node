package peerchoice_test

import (
	"reflect"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheChosenPeersOfEachPartitionKeepTheirOrderInTheOrderOfTheirFirstPeer(t *testing.T) {
	t.Parallel()

	first, second, third := peerNamed("first"), peerNamed("second"), peerNamed("third")
	chosenPeersOfQueryWord := peerchoice.ChosenPeersOfQueryWord{
		QueryWord: yacymodel.WordHash("berlin"),
		ChosenPeers: []peerchoice.ChosenPeer{
			{Peer: second, Partition: 7},
			{Peer: third, Partition: 2},
			{Peer: first, Partition: 7},
		},
	}

	wanted := []peerchoice.ChosenPeersOfPartition{
		{Partition: 7, Peers: []peerdirectory.AskablePeer{second, first}},
		{Partition: 2, Peers: []peerdirectory.AskablePeer{third}},
	}
	if got := chosenPeersOfQueryWord.ChosenPeersPerPartition(); !reflect.DeepEqual(got, wanted) {
		t.Fatalf("ChosenPeersPerPartition = %+v, want %+v", got, wanted)
	}
}

func peerNamed(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}
