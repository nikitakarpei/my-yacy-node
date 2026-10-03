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

type chosenReplicas struct {
	query                    searchquery.Query
	replicasPerWordPartition map[wordPartitionKey][]peerdirectory.AskablePeer
}

func chosenReplicasFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) chosenReplicas {
	replicasPerWordPartition := map[wordPartitionKey][]peerdirectory.AskablePeer{}
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeersOfPartition := range chosenPeersOfQueryWord.ChosenPeersPerPartition() {
			key := wordPartitionKey{
				word:      chosenPeersOfQueryWord.QueryWord,
				partition: chosenPeersOfPartition.Partition,
			}
			replicasPerWordPartition[key] = chosenPeersOfPartition.Peers
		}
	}

	return chosenReplicas{query: query, replicasPerWordPartition: replicasPerWordPartition}
}

func (replicas chosenReplicas) asksOfEveryPartitionFor(
	words []yacymodel.Hash,
	partitions yacymodel.DHTRingPartitions,
) []wordpartitionasks.Ask {
	asks := make([]wordpartitionasks.Ask, 0, len(words)*int(partitions))
	for _, word := range words {
		for partition := range uint(partitions) {
			asks = append(asks, replicas.asksOf([]yacymodel.Hash{word}, partition)...)
		}
	}

	return asks
}

func (replicas chosenReplicas) asksOf(
	words []yacymodel.Hash,
	partition uint,
) []wordpartitionasks.Ask {
	var asks []wordpartitionasks.Ask
	for _, word := range words {
		replicasInOrder, chosen := replicas.replicasPerWordPartition[wordPartitionKey{
			word: word, partition: partition,
		}]
		if !chosen {
			continue
		}
		asks = append(asks, wordpartitionasks.Ask{
			Word:            word,
			Partition:       partition,
			ReplicasInOrder: replicasInOrder,
			ExcludedWords:   replicas.query.ExclusionHashes(),
			Language:        replicas.query.Language,
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
