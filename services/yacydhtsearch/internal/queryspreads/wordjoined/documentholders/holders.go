// Package documentholders keeps which peers listed each document in their
// answers, which documents each peer listed, and orders documents by how many
// peers hold them.
package documentholders

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Holders struct {
	holdersPerDocument          map[yacymodel.URLHash]map[yacymodel.Hash]struct{}
	peerWithItsDocumentsPerPeer map[yacymodel.Hash]PeerWithItsDocuments
}

type PeerWithItsDocuments struct {
	Peer      peerdirectory.AskablePeer
	Documents yacymodel.URLHashes
}

func NoHolders() Holders {
	return Holders{
		holdersPerDocument:          map[yacymodel.URLHash]map[yacymodel.Hash]struct{}{},
		peerWithItsDocumentsPerPeer: map[yacymodel.Hash]PeerWithItsDocuments{},
	}
}

func (holders Holders) AddHoldersIn(answers []wordpartitionasks.ReplicaAnswer) {
	for _, answer := range answers {
		peer := holders.peerWithItsDocumentsFor(answer.Replica)
		for _, listedDocument := range answer.ListedDocuments {
			if holders.holdersPerDocument[listedDocument.Hash] == nil {
				holders.holdersPerDocument[listedDocument.Hash] = map[yacymodel.Hash]struct{}{}
			}
			holders.holdersPerDocument[listedDocument.Hash][answer.Replica.Hash] = struct{}{}
			peer.Documents.Add(listedDocument.Hash)
		}
	}
}

func (holders Holders) peerWithItsDocumentsFor(
	replica peerdirectory.AskablePeer,
) PeerWithItsDocuments {
	peer, known := holders.peerWithItsDocumentsPerPeer[replica.Hash]
	if !known {
		peer = PeerWithItsDocuments{Peer: replica, Documents: yacymodel.URLHashes{}}
		holders.peerWithItsDocumentsPerPeer[replica.Hash] = peer
	}

	return peer
}

func (holders Holders) MostHeldFirst(documents yacymodel.URLHashes) []yacymodel.URLHash {
	return slices.SortedFunc(
		maps.Keys(documents),
		func(first, second yacymodel.URLHash) int {
			amountOfHoldersOfFirst := len(holders.holdersPerDocument[first])
			amountOfHoldersOfSecond := len(holders.holdersPerDocument[second])
			if amountOfHoldersOfFirst != amountOfHoldersOfSecond {
				return cmp.Compare(amountOfHoldersOfSecond, amountOfHoldersOfFirst)
			}

			return strings.Compare(first.String(), second.String())
		},
	)
}

func (holders Holders) PeersWithTheirDocuments() []PeerWithItsDocuments {
	return slices.SortedFunc(
		maps.Values(holders.peerWithItsDocumentsPerPeer),
		func(first, second PeerWithItsDocuments) int {
			return strings.Compare(first.Peer.Hash.String(), second.Peer.Hash.String())
		},
	)
}
