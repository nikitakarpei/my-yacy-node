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
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	partitions yacymodel.DHTRingPartitions,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []queryWordAcrossReplicas {
	queryWordsAcrossReplicas := make([]queryWordAcrossReplicas, 0, len(chosenPeersPerQueryWord))
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		queryWordsAcrossReplicas = append(queryWordsAcrossReplicas, queryWordAcrossReplicas{
			word: chosenPeersOfQueryWord.QueryWord,
			queryWordOnReplicasPerPartition: queryWordOnReplicasPerPartitionFrom(
				chosenPeersOfQueryWord, partitions, answeredAsks,
			),
		})
	}
	slices.SortStableFunc(queryWordsAcrossReplicas, fewestDocumentsFirst)

	return queryWordsAcrossReplicas
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
				answer: answerOfPeerFor(
					chosenPeer.Peer,
					chosenPeersOfQueryWord.QueryWord,
					answeredAsks,
				),
			},
		)
	}

	return queryWordOnReplicasPerPartition
}

func answerOfPeerFor(
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
	amountsHeldInCountedPartitions := queryWord.amountsOfDocumentsHeldInCountedPartitions()
	if len(amountsHeldInCountedPartitions) == 0 {
		return yacymodel.None[int]()
	}

	amountOfUncountedPartitions := len(
		queryWord.queryWordOnReplicasPerPartition,
	) - len(
		amountsHeldInCountedPartitions,
	)
	sumOfAmountsHeld := amountOfUncountedPartitions * lowerMedianOf(amountsHeldInCountedPartitions)
	for _, amountHeld := range amountsHeldInCountedPartitions {
		sumOfAmountsHeld += amountHeld
	}

	return yacymodel.Some(sumOfAmountsHeld)
}

func (queryWord queryWordAcrossReplicas) amountsOfDocumentsHeldInCountedPartitions() []int {
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

func (queryWord queryWordAcrossReplicas) isFullyListed() bool {
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		if !slices.ContainsFunc(queryWordOnReplicasOfPartition, queryWordOnReplica.isFullyListed) {
			return false
		}
	}

	return true
}

func (queryWord queryWordAcrossReplicas) peersThatDidNotListAllTheyHold() []peerdirectory.AskablePeer {
	var peers []peerdirectory.AskablePeer
	for _, queryWordOnReplicasOfPartition := range queryWord.queryWordOnReplicasPerPartition {
		for _, queryWordOnOneReplica := range queryWordOnReplicasOfPartition {
			if queryWordOnOneReplica.isFullyListed() {
				continue
			}
			peers = append(peers, queryWordOnOneReplica.peer)
		}
	}

	return peers
}

func (queryWord queryWordOnReplica) isFullyListed() bool {
	answer, answered := queryWord.answer.Get()
	if !answered {
		return false
	}
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountOfDocumentsHeld <= len(answer.DocumentsListedForTheWord)
}
