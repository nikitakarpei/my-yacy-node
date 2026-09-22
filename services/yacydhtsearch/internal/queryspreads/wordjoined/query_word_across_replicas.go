package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type queryWordAcrossReplicas struct {
	word                            yacymodel.Hash
	queryWordOnReplicasPerPartition [][]queryWordOnReplica
}

type queryWordOnReplica struct {
	peer   peerdirectory.AskablePeer
	answer yacymodel.Optional[peerasks.AnsweredMatchedAndHeldDocumentsAsk]
}

func queryWordsFewestDocumentsFirstFrom(
	words []yacymodel.Hash,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
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
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) queryWordAcrossReplicas {
	return queryWordAcrossReplicas{
		word: chosenPeersOfQueryWord.QueryWord,
		queryWordOnReplicasPerPartition: queryWordOnReplicasPerPartitionFrom(
			chosenPeersOfQueryWord, partitions, answeredAsks,
		),
	}
}

func queryWordOnReplicasPerPartitionFrom(
	chosenPeersOfQueryWord peerchoice.ChosenPeersOfQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) [][]queryWordOnReplica {
	queryWordOnReplicasPerPartition := make([][]queryWordOnReplica, partitions)
	for _, chosenPeer := range chosenPeersOfQueryWord.ChosenPeers {
		answer := yacymodel.None[peerasks.AnsweredMatchedAndHeldDocumentsAsk]()
		place := slices.IndexFunc(
			answeredAsks,
			func(answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk) bool {
				return answeredAsk.AnswersTheAskTo(
					chosenPeer.Peer,
					chosenPeersOfQueryWord.QueryWord,
				)
			},
		)
		if place >= 0 {
			answer = yacymodel.Some(answeredAsks[place])
		}
		queryWordOnReplicasPerPartition[chosenPeer.Partition] = append(
			queryWordOnReplicasPerPartition[chosenPeer.Partition],
			queryWordOnReplica{peer: chosenPeer.Peer, answer: answer},
		)
	}

	return queryWordOnReplicasPerPartition
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

	amountOfPartitionsWhereNoPeerCounted := len(queryWord.queryWordOnReplicasPerPartition) -
		len(amountsHeldInPartitionsWhereAPeerCounted)
	sumOfAmountsHeld := amountOfPartitionsWhereNoPeerCounted *
		lowerMedianOf(amountsHeldInPartitionsWhereAPeerCounted)
	for _, amountHeld := range amountsHeldInPartitionsWhereAPeerCounted {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (queryWord queryWordAcrossReplicas) amountsOfDocumentsHeldInPartitionsWhereAPeerCounted() []int {
	amountsHeld := make([]int, 0, len(queryWord.queryWordOnReplicasPerPartition))
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		countedAmounts := amountsOfDocumentsHeldCountedBy(queryWordOnReplicasOfPartition)
		if len(countedAmounts) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(countedAmounts))
	}

	return amountsHeld
}

func amountsOfDocumentsHeldCountedBy(queryWordOnReplicas []queryWordOnReplica) []int {
	countedAmounts := make([]int, 0, len(queryWordOnReplicas))
	for _, queryWordOnOneReplica := range queryWordOnReplicas {
		answer, answered := queryWordOnOneReplica.answer.Get()
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
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		for _, queryWordOnOneReplica := range queryWordOnReplicasOfPartition {
			answer, answered := queryWordOnOneReplica.answer.Get()
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
	wordPartitions := make([]wordPartition, 0, len(queryWord.queryWordOnReplicasPerPartition))
	for _, replicas := range queryWord.queryWordOnReplicasPerPartition {
		wordPartitions = append(
			wordPartitions,
			wordPartition{word: queryWord.word, replicas: replicas},
		)
	}

	return wordPartitions
}

func (queryWord queryWordAcrossReplicas) replicasThatDidNotListAllTheyHold() []queryWordOnReplica {
	var replicas []queryWordOnReplica
	for _, wordPartition := range queryWord.wordPartitions() {
		replicas = append(replicas, wordPartition.replicasThatDidNotListAllTheyHold()...)
	}

	return replicas
}

func (replica queryWordOnReplica) isFullyListed() bool {
	answer, answered := replica.answer.Get()
	if !answered {
		return false
	}
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountOfDocumentsHeld <= len(answer.DocumentsListedForTheWord)
}

func (replica queryWordOnReplica) versionClaimed() string {
	answer, answered := replica.answer.Get()
	if !answered {
		return ""
	}

	return answer.PeerVersion
}
