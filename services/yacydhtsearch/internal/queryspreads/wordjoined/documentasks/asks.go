package documentasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

type chosenPeers struct {
	query                 searchquery.Query
	peersPerWordPartition map[wordPartitionKey][]peerdirectory.AskablePeer
}

func chosenPeersFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) chosenPeers {
	peersPerWordPartition := map[wordPartitionKey][]peerdirectory.AskablePeer{}
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeersOfPartition := range chosenPeersOfQueryWord.ChosenPeersPerPartition() {
			key := wordPartitionKey{
				word:      chosenPeersOfQueryWord.QueryWord,
				partition: chosenPeersOfPartition.Partition,
			}
			peersPerWordPartition[key] = chosenPeersOfPartition.Peers
		}
	}

	return chosenPeers{query: query, peersPerWordPartition: peersPerWordPartition}
}

func (peers chosenPeers) asksOfEveryPartitionFor(
	words []yacymodel.Hash,
	partitions yacymodel.DHTRingPartitions,
) []wordpartitionasks.Ask {
	asks := make([]wordpartitionasks.Ask, 0, len(words)*int(partitions))
	for _, word := range words {
		for partition := range uint(partitions) {
			asks = append(asks, peers.asksOf([]yacymodel.Hash{word}, partition)...)
		}
	}

	return asks
}

func (peers chosenPeers) asksOf(
	words []yacymodel.Hash,
	partition uint,
) []wordpartitionasks.Ask {
	var asks []wordpartitionasks.Ask
	for _, word := range words {
		peersInOrder, chosen := peers.peersPerWordPartition[wordPartitionKey{
			word: word, partition: partition,
		}]
		if !chosen {
			continue
		}
		asks = append(asks, wordpartitionasks.Ask{
			Word:            word,
			Partition:       partition,
			ReplicasInOrder: peersInOrder,
			ExcludedWords:   peers.query.ExclusionHashes(),
			Language:        peers.query.Language,
		})
	}

	return asks
}

func withDocumentsToMatch(
	asks []wordpartitionasks.Ask,
	documentsToMatch []yacymodel.URLHash,
) []wordpartitionasks.Ask {
	asksForTheDocuments := make([]wordpartitionasks.Ask, 0, len(asks))
	for _, ask := range asks {
		ask.DocumentsToMatch = documentsToMatch
		asksForTheDocuments = append(asksForTheDocuments, ask)
	}

	return asksForTheDocuments
}
