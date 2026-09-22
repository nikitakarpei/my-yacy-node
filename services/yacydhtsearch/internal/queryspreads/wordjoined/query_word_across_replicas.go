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
		queryWordOnReplicasPerPartition[chosenPeer.Partition] = append(
			queryWordOnReplicasPerPartition[chosenPeer.Partition],
			queryWordOnReplica{
				peer: chosenPeer.Peer,
				answer: answerToTheMatchedAndHeldDocumentsAskAmong(
					chosenPeer.Peer,
					chosenPeersOfQueryWord.QueryWord,
					answeredAsks,
				),
			},
		)
	}

	return queryWordOnReplicasPerPartition
}

func answerToTheMatchedAndHeldDocumentsAskAmong(
	peer peerdirectory.AskablePeer,
	queryWord yacymodel.Hash,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) yacymodel.Optional[peerasks.AnsweredMatchedAndHeldDocumentsAsk] {
	place := slices.IndexFunc(
		answeredAsks,
		func(answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk) bool {
			return answeredAsk.Ask.Peer.Hash == peer.Hash && answeredAsk.Ask.Word == queryWord
		},
	)
	if place < 0 {
		return yacymodel.None[peerasks.AnsweredMatchedAndHeldDocumentsAsk]()
	}

	return yacymodel.Some(answeredAsks[place])
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

func (queryWord queryWordAcrossReplicas) crossCheckCandidatesAmong(
	documentsOfTheLeadingQueryWord []yacymodel.URLHash,
) []yacymodel.URLHash {
	documentsListedByPeers := queryWord.documentsListedByPeers()
	candidates := make([]yacymodel.URLHash, 0, len(documentsOfTheLeadingQueryWord))
	for _, document := range documentsOfTheLeadingQueryWord {
		if documentsListedByPeers.contains(document) {
			continue
		}
		candidates = append(candidates, document)
	}

	return candidates
}

func (queryWord queryWordAcrossReplicas) isFullyListed() bool {
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		if !slices.ContainsFunc(queryWordOnReplicasOfPartition, queryWordOnReplica.isFullyListed) {
			return false
		}
	}

	return true
}

func (queryWord queryWordAcrossReplicas) replicasThatDidNotListAllTheyHold() []queryWordOnReplica {
	var replicas []queryWordOnReplica
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		for _, queryWordOnOneReplica := range queryWordOnReplicasOfPartition {
			if queryWordOnOneReplica.isFullyListed() {
				continue
			}
			replicas = append(replicas, queryWordOnOneReplica)
		}
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
