package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type answeredQueryWord struct {
	word       yacymodel.Hash
	partitions yacymodel.DHTRingPartitions
	replicas   []replicaOfQueryWord
}

type replicaOfQueryWord struct {
	peer      peerdirectory.AskablePeer
	partition uint
	answer    yacymodel.Optional[peerasks.AnsweredMatchedAndHeldDocumentsAsk]
}

func queryWordsFewestDocumentsFirstFrom(
	queryWords []yacymodel.Hash,
	chosenPeersPerQueryWord []peerchoice.PeersOfQueryWord,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []answeredQueryWord {
	answerOfEachReplica := answerOfEachReplicaOf(answeredAsks)
	answeredQueryWords := make([]answeredQueryWord, 0, len(queryWords))
	for index, queryWord := range queryWords {
		answeredQueryWords = append(answeredQueryWords, answeredQueryWord{
			word:       queryWord,
			partitions: chosenPeersPerQueryWord[index].Partitions,
			replicas: replicasOfQueryWordFrom(
				queryWord, chosenPeersPerQueryWord[index], answerOfEachReplica,
			),
		})
	}
	slices.SortStableFunc(answeredQueryWords, fewestDocumentsFirst)

	return answeredQueryWords
}

type peerOfQueryWord struct {
	peer yacymodel.Hash
	word yacymodel.Hash
}

func answerOfEachReplicaOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) map[peerOfQueryWord]peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	answerOfEachReplica := make(
		map[peerOfQueryWord]peerasks.AnsweredMatchedAndHeldDocumentsAsk, len(answeredAsks),
	)
	for _, answeredAsk := range answeredAsks {
		answerOfEachReplica[peerOfQueryWord{
			peer: answeredAsk.Ask.Peer.Hash, word: answeredAsk.Ask.Word,
		}] = answeredAsk
	}

	return answerOfEachReplica
}

func replicasOfQueryWordFrom(
	queryWord yacymodel.Hash,
	chosenPeers peerchoice.PeersOfQueryWord,
	answerOfEachReplica map[peerOfQueryWord]peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []replicaOfQueryWord {
	replicas := make([]replicaOfQueryWord, 0, len(chosenPeers.ChosenPeers))
	for _, chosenPeer := range chosenPeers.ChosenPeers {
		answer := yacymodel.None[peerasks.AnsweredMatchedAndHeldDocumentsAsk]()
		if answeredAsk, answered := answerOfEachReplica[peerOfQueryWord{
			peer: chosenPeer.Peer.Hash, word: queryWord,
		}]; answered {
			answer = yacymodel.Some(answeredAsk)
		}
		replicas = append(replicas, replicaOfQueryWord{
			peer:      chosenPeer.Peer,
			partition: chosenPeer.Partition,
			answer:    answer,
		})
	}

	return replicas
}

func fewestDocumentsFirst(first, second answeredQueryWord) int {
	documentsOfTheFirst, countedForTheFirst := first.amountOfDocumentsHeld()
	documentsOfTheSecond, countedForTheSecond := second.amountOfDocumentsHeld()
	if countedForTheFirst != countedForTheSecond {
		if countedForTheFirst {
			return -1
		}

		return 1
	}

	return cmp.Compare(documentsOfTheFirst, documentsOfTheSecond)
}

func (queryWord answeredQueryWord) amountOfDocumentsHeld() (int, bool) {
	amountsHeldInCountedPartitions := queryWord.amountsOfDocumentsHeldInCountedPartitions()
	if len(amountsHeldInCountedPartitions) == 0 {
		return 0, false
	}

	amountOfUncountedPartitions := int(queryWord.partitions) - len(amountsHeldInCountedPartitions)
	sumOfAmountsHeld := amountOfUncountedPartitions * lowerMedianOf(amountsHeldInCountedPartitions)
	for _, amountHeld := range amountsHeldInCountedPartitions {
		sumOfAmountsHeld += amountHeld
	}

	return sumOfAmountsHeld, true
}

func (queryWord answeredQueryWord) amountsOfDocumentsHeldInCountedPartitions() []int {
	countedAmountsPerPartition := make([][]int, queryWord.partitions)
	for _, replica := range queryWord.replicas {
		answer, answered := replica.answer.Get()
		if !answered {
			continue
		}
		amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		countedAmountsPerPartition[replica.partition] = append(
			countedAmountsPerPartition[replica.partition], amountOfDocumentsHeld,
		)
	}

	amountsHeld := make([]int, 0, len(countedAmountsPerPartition))
	for _, countedAmounts := range countedAmountsPerPartition {
		if len(countedAmounts) == 0 {
			continue
		}
		amountsHeld = append(amountsHeld, lowerMedianOf(countedAmounts))
	}

	return amountsHeld
}

func lowerMedianOf(amounts []int) int {
	sortedAmounts := slices.Sorted(slices.Values(amounts))

	return sortedAmounts[(len(sortedAmounts)-1)/2]
}

func (queryWord answeredQueryWord) documentsHeld() map[yacymodel.URLHash]struct{} {
	documentsHeld := map[yacymodel.URLHash]struct{}{}
	for _, replica := range queryWord.replicas {
		answer, answered := replica.answer.Get()
		if !answered {
			continue
		}
		for _, document := range answer.DocumentsHeldForTheWord {
			documentsHeld[document] = struct{}{}
		}
	}

	return documentsHeld
}

func (queryWord answeredQueryWord) isComplete() bool {
	partitionsWithACompleteAnswer := map[uint]struct{}{}
	for _, replica := range queryWord.replicas {
		if !replica.listedAllItHolds() {
			continue
		}
		partitionsWithACompleteAnswer[replica.partition] = struct{}{}
	}

	return len(partitionsWithACompleteAnswer) >= int(queryWord.partitions)
}

func (queryWord answeredQueryWord) cutOffOrSilentPeers() []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(queryWord.replicas))
	for _, replica := range queryWord.replicas {
		if replica.listedAllItHolds() {
			continue
		}
		peers = append(peers, replica.peer)
	}

	return peers
}

func (replica replicaOfQueryWord) listedAllItHolds() bool {
	answer, answered := replica.answer.Get()
	if !answered {
		return false
	}
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountOfDocumentsHeld <= len(answer.DocumentsHeldForTheWord)
}
