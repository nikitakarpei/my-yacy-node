package documentasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

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
			if ask, chosen := peers.askOf(word, partition).Get(); chosen {
				asks = append(asks, ask)
			}
		}
	}

	return asks
}

func (peers chosenPeers) asksOf(words []yacymodel.Hash, partition uint) []wordpartitionasks.Ask {
	var asks []wordpartitionasks.Ask
	for _, word := range words {
		if ask, chosen := peers.askOf(word, partition).Get(); chosen {
			asks = append(asks, ask)
		}
	}

	return asks
}

func (peers chosenPeers) askOf(
	word yacymodel.Hash,
	partition uint,
) yacymodel.Optional[wordpartitionasks.Ask] {
	peersInOrder, chosen := peers.peersPerWordPartition[wordPartitionKey{word: word, partition: partition}]
	if !chosen {
		return yacymodel.None[wordpartitionasks.Ask]()
	}

	return yacymodel.Some(wordpartitionasks.Ask{
		Word:            word,
		Partition:       partition,
		ReplicasInOrder: peersInOrder,
		ExcludedWords:   peers.query.ExclusionHashes(),
		Language:        peers.query.Language,
	})
}

func namingTheDocumentsToMatch(
	asks []wordpartitionasks.Ask,
	documentsToMatch []yacymodel.URLHash,
) []wordpartitionasks.Ask {
	asksNamingTheDocuments := make([]wordpartitionasks.Ask, 0, len(asks))
	for _, ask := range asks {
		ask.DocumentsToMatch = documentsToMatch
		asksNamingTheDocuments = append(asksNamingTheDocuments, ask)
	}

	return asksNamingTheDocuments
}
