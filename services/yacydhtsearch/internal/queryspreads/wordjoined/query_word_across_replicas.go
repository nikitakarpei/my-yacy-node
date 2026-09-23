package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordAcrossReplicas struct {
	word                 yacymodel.Hash
	replicasPerPartition [][]wordReplica
}

func queryWordsFewestDocumentsFirstFrom(
	words []yacymodel.Hash,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) []queryWordAcrossReplicas {
	queryWordsAcrossReplicas := make([]queryWordAcrossReplicas, 0, len(words))
	for _, word := range words {
		chosenPeersOfWord, _ := chosenPeersOf(word, chosenPeersPerQueryWord)
		queryWordsAcrossReplicas = append(
			queryWordsAcrossReplicas,
			queryWordAcrossReplicasFrom(chosenPeersOfWord, partitions, answeredAsks),
		)
	}
	slices.SortStableFunc(queryWordsAcrossReplicas, fewestDocumentsFirst)

	return queryWordsAcrossReplicas
}

func chosenPeersOf(
	word yacymodel.Hash,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) (peerchoice.ChosenPeersOfQueryWord, bool) {
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		if chosenPeersOfQueryWord.QueryWord == word {
			return chosenPeersOfQueryWord, true
		}
	}

	return peerchoice.ChosenPeersOfQueryWord{QueryWord: word}, false
}

func queryWordAcrossReplicasFrom(
	chosenPeersOfQueryWord peerchoice.ChosenPeersOfQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) queryWordAcrossReplicas {
	return queryWordAcrossReplicas{
		word: chosenPeersOfQueryWord.QueryWord,
		replicasPerPartition: replicasPerPartitionFrom(
			chosenPeersOfQueryWord, partitions, answeredAsks,
		),
	}
}

func replicasPerPartitionFrom(
	chosenPeersOfQueryWord peerchoice.ChosenPeersOfQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) [][]wordReplica {
	replicasPerPartition := make([][]wordReplica, partitions)
	for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
		answer := yacymodel.None[peerasks.AnsweredSearchDocumentsAsk]()
		place := slices.IndexFunc(
			answeredAsks,
			func(answeredAsk peerasks.AnsweredSearchDocumentsAsk) bool {
				return answeredAsk.AnswersTheAskTo(
					chosenPeer.Peer,
					chosenPeersOfQueryWord.QueryWord,
				)
			},
		)
		if place >= 0 {
			answer = yacymodel.Some(answeredAsks[place])
		}
		replicasPerPartition[chosenPeer.Partition] = append(
			replicasPerPartition[chosenPeer.Partition],
			wordReplica{peer: chosenPeer.Peer, answer: answer},
		)
	}

	return replicasPerPartition
}

func fewestDocumentsFirst(first, second queryWordAcrossReplicas) int {
	documentsOfTheFirst, countedForTheFirst := first.estimatedAmountOfDocumentsHeld().Get()
	documentsOfTheSecond, countedForTheSecond := second.estimatedAmountOfDocumentsHeld().Get()
	if countedForTheFirst != countedForTheSecond {
		if countedForTheFirst {
			return -1
		}

		return 1
	}

	return cmp.Compare(documentsOfTheFirst, documentsOfTheSecond)
}

func (queryWord queryWordAcrossReplicas) estimatedAmountOfDocumentsHeld() yacymodel.Optional[int] {
	amountsHeldInPartitionsWhereAPeerCounted := queryWord.amountsOfDocumentsHeldInPartitionsWhereAPeerCounted()
	if len(amountsHeldInPartitionsWhereAPeerCounted) == 0 {
		return yacymodel.None[int]()
	}

	amountOfPartitionsWhereNoPeerCounted := len(queryWord.replicasPerPartition) -
		len(amountsHeldInPartitionsWhereAPeerCounted)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted *
		lowerMedianOf(amountsHeldInPartitionsWhereAPeerCounted)
	for _, amountHeld := range amountsHeldInPartitionsWhereAPeerCounted {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (queryWord queryWordAcrossReplicas) amountsOfDocumentsHeldInPartitionsWhereAPeerCounted() []int {
	amountsHeld := make([]int, 0, len(queryWord.replicasPerPartition))
	for _, replicasOfPartition := range queryWord.replicasPerPartition {
		countedAmounts := amountsOfDocumentsHeldCountedBy(replicasOfPartition)
		if len(countedAmounts) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(countedAmounts))
	}

	return amountsHeld
}

func amountsOfDocumentsHeldCountedBy(replicas []wordReplica) []int {
	countedAmounts := make([]int, 0, len(replicas))
	for _, replica := range replicas {
		answer, answered := replica.answer.Get()
		if !answered {
			continue
		}
		amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		countedAmounts = append(countedAmounts, amountOfDocumentsHeld)
	}

	return countedAmounts
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (queryWord queryWordAcrossReplicas) documentsListedByPeers() distinctDocuments {
	documentsListedByPeers := distinctDocuments{}
	for _, replicasOfPartition := range queryWord.replicasPerPartition {
		for _, replica := range replicasOfPartition {
			answer, answered := replica.answer.Get()
			if !answered {
				continue
			}
			for _, document := range answer.DocumentsListedForTheWord {
				documentsListedByPeers.add(document)
			}
		}
	}

	return documentsListedByPeers
}

func (queryWord queryWordAcrossReplicas) documentsNotListedByItsPeersAmong(
	documents []yacymodel.URLHash,
) []yacymodel.URLHash {
	documentsListedByPeers := queryWord.documentsListedByPeers()
	documentsNotListed := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		if documentsListedByPeers.contains(document) {
			continue
		}
		documentsNotListed = append(documentsNotListed, document)
	}

	return documentsNotListed
}

func (queryWord queryWordAcrossReplicas) isFullyListed() bool {
	for _, wordPartition := range queryWord.wordPartitions() {
		if !wordPartition.isFullyListed() {
			return false
		}
	}

	return true
}

func (queryWord queryWordAcrossReplicas) wordPartitions() []wordPartition {
	wordPartitions := make([]wordPartition, 0, len(queryWord.replicasPerPartition))
	for partition, replicas := range queryWord.replicasPerPartition {
		wordPartitions = append(
			wordPartitions,
			wordPartition{word: queryWord.word, partition: uint(partition), replicas: replicas},
		)
	}

	return wordPartitions
}
