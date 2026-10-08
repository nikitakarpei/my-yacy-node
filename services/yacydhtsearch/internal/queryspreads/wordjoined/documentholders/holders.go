// Package documentholders keeps which peers listed each document in their
// answers and which documents came with metadata. The holders of some documents
// name the peers that hold them and order the documents by how many peers hold
// them.
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
	peersPerDocument      map[yacymodel.URLHash][]peerdirectory.AskablePeer
	documentsWithMetadata yacymodel.URLHashes
}

func NoneYet() Holders {
	return Holders{
		peersPerDocument:      map[yacymodel.URLHash][]peerdirectory.AskablePeer{},
		documentsWithMetadata: yacymodel.URLHashes{},
	}
}

func (holders Holders) WordPartitionAnswered(
	_ yacymodel.Hash,
	_ uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			holders.keep(answer.Replica, listedDocument)
		}
	}
}

func (holders Holders) keep(
	replica peerdirectory.AskablePeer,
	listedDocument wordpartitionasks.ListedDocument,
) {
	peers := holders.peersPerDocument[listedDocument.Hash]
	if !slices.ContainsFunc(peers, func(peer peerdirectory.AskablePeer) bool {
		return peer.Hash == replica.Hash
	}) {
		holders.peersPerDocument[listedDocument.Hash] = append(peers, replica)
	}
	if listedDocument.Metadata().Present() {
		holders.documentsWithMetadata.Add(listedDocument.Hash)
	}
}

func (holders Holders) HoldersOf(documents yacymodel.URLHashes) Holders {
	holdersOfTheDocuments := NoneYet()
	for document := range documents {
		peers, listed := holders.peersPerDocument[document]
		if !listed {
			continue
		}
		holdersOfTheDocuments.peersPerDocument[document] = slices.Clone(peers)
	}

	return holdersOfTheDocuments
}

func (holders Holders) WithoutMetadataAmong(documents yacymodel.URLHashes) yacymodel.URLHashes {
	documentsWithoutMetadata := yacymodel.URLHashes{}
	for document := range documents {
		if holders.documentsWithMetadata.Contains(document) {
			continue
		}
		documentsWithoutMetadata.Add(document)
	}

	return documentsWithoutMetadata
}

func (holders Holders) LeastHeldFirst() []yacymodel.URLHash {
	return slices.SortedFunc(
		maps.Keys(holders.peersPerDocument),
		func(first, second yacymodel.URLHash) int {
			amountOfHoldersOfFirst := len(holders.peersPerDocument[first])
			amountOfHoldersOfSecond := len(holders.peersPerDocument[second])
			if amountOfHoldersOfFirst != amountOfHoldersOfSecond {
				return cmp.Compare(amountOfHoldersOfFirst, amountOfHoldersOfSecond)
			}

			return strings.Compare(first.String(), second.String())
		},
	)
}

func (holders Holders) PeersHolding(document yacymodel.URLHash) []peerdirectory.AskablePeer {
	return slices.Clone(holders.peersPerDocument[document])
}

func (holders Holders) Peers() []peerdirectory.AskablePeer {
	peerOfEachHash := map[yacymodel.Hash]peerdirectory.AskablePeer{}
	for _, peersOfTheDocument := range holders.peersPerDocument {
		for _, peer := range peersOfTheDocument {
			peerOfEachHash[peer.Hash] = peer
		}
	}

	return slices.Collect(maps.Values(peerOfEachHash))
}
