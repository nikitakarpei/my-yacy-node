package peerchoice

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type peersOneQueryMayAsk struct {
	partitions            yacymodel.DHTRingPartitions
	networkRedundancy     int
	askablePeers          []peerdirectory.AskablePeer
	reliabilityOfEachPeer map[yacymodel.Hash]float64
}

func (q peersOneQueryMayAsk) peersForQueryWord(
	queryWord yacymodel.Hash,
	peersChosenForEarlierWords []peerdirectory.AskablePeer,
) (
	chosenPeers []ChosenPeer,
	ringFractionsOfTheTakenPeers []float64,
) {
	return peersTakenFromEachPartitionInTurn(
		q.peersNearestToTheWordInEachPartition(queryWord, peersChosenForEarlierWords),
		q.networkRedundancy,
	)
}

type peerAtRingFractionFromTheWord struct {
	peer                    peerdirectory.AskablePeer
	ringFractionFromTheWord float64
}

func (q peersOneQueryMayAsk) peersNearestToTheWordInEachPartition(
	queryWord yacymodel.Hash,
	peersChosenForEarlierWords []peerdirectory.AskablePeer,
) [][]peerAtRingFractionFromTheWord {
	peersNotChosenForEarlierWords := peersNotChosenForEarlierWordsAmong(
		q.askablePeers, peersChosenForEarlierWords,
	)
	nearestToTheWordInEachPartition := make([][]peerAtRingFractionFromTheWord, 0, q.partitions)
	for partition := range uint(q.partitions) {
		wordPosition := yacymodel.DHTRingPositionOfWordInPartition(
			queryWord, partition, q.partitions,
		)
		nearestToTheWordInEachPartition = append(
			nearestToTheWordInEachPartition,
			slices.Concat(
				q.reliablePeersFirstAmongThePeersHoldingTheWord(
					peersNearestFirstTo(wordPosition, peersNotChosenForEarlierWords),
				),
				peersNearestFirstTo(wordPosition, peersChosenForEarlierWords),
			),
		)
	}

	return nearestToTheWordInEachPartition
}

func peersNotChosenForEarlierWordsAmong(
	askablePeers []peerdirectory.AskablePeer,
	peersChosenForEarlierWords []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	hashesOfPeersChosenForEarlierWords := hashesOf(peersChosenForEarlierWords)

	return slices.DeleteFunc(
		slices.Clone(askablePeers),
		func(peer peerdirectory.AskablePeer) bool {
			_, chosen := hashesOfPeersChosenForEarlierWords[peer.Hash]

			return chosen
		},
	)
}

func hashesOf(peers []peerdirectory.AskablePeer) map[yacymodel.Hash]struct{} {
	hashes := make(map[yacymodel.Hash]struct{}, len(peers))
	for _, peer := range peers {
		hashes[peer.Hash] = struct{}{}
	}

	return hashes
}

func (q peersOneQueryMayAsk) reliablePeersFirstAmongThePeersHoldingTheWord(
	peersNearestFirst []peerAtRingFractionFromTheWord,
) []peerAtRingFractionFromTheWord {
	amountOfPeersHoldingTheWordInOnePartition := min(len(peersNearestFirst), q.networkRedundancy)

	return slices.Concat(
		slices.SortedStableFunc(
			slices.Values(peersNearestFirst[:amountOfPeersHoldingTheWordInOnePartition]),
			func(firstPeer, secondPeer peerAtRingFractionFromTheWord) int {
				return cmp.Compare(
					q.reliabilityOfEachPeer[secondPeer.peer.Hash],
					q.reliabilityOfEachPeer[firstPeer.peer.Hash],
				)
			},
		),
		peersNearestFirst[amountOfPeersHoldingTheWordInOnePartition:],
	)
}

func peersNearestFirstTo(
	wordPosition yacymodel.DHTRingPosition,
	peers []peerdirectory.AskablePeer,
) []peerAtRingFractionFromTheWord {
	peersAtRingFractionFromTheWord := make([]peerAtRingFractionFromTheWord, 0, len(peers))
	for _, peer := range peers {
		peersAtRingFractionFromTheWord = append(
			peersAtRingFractionFromTheWord,
			peerAtRingFractionFromTheWord{
				peer:                    peer,
				ringFractionFromTheWord: ringFractionFrom(wordPosition, peer),
			},
		)
	}
	slices.SortFunc(
		peersAtRingFractionFromTheWord,
		func(firstPeer, secondPeer peerAtRingFractionFromTheWord) int {
			return cmp.Compare(
				firstPeer.ringFractionFromTheWord,
				secondPeer.ringFractionFromTheWord,
			)
		},
	)

	return peersAtRingFractionFromTheWord
}

func ringFractionFrom(
	wordPosition yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
) float64 {
	return wordPosition.DistanceTo(
		yacymodel.DHTRingPositionOf(peer.Hash),
	).FractionOfDHTRing()
}

func peersTakenFromEachPartitionInTurn(
	peersNearestToTheWordInEachPartition [][]peerAtRingFractionFromTheWord,
	networkRedundancy int,
) (takenPeers []ChosenPeer, ringFractionsOfTheTakenPeers []float64) {
	peersAlreadyTaken := map[yacymodel.Hash]struct{}{}
	for range networkRedundancy {
		for index, peersNearestFirst := range peersNearestToTheWordInEachPartition {
			takenPeer, peersLeft, found := firstPeerNotYetTaken(
				peersNearestFirst,
				peersAlreadyTaken,
			)
			if !found {
				return takenPeers, ringFractionsOfTheTakenPeers
			}
			peersNearestToTheWordInEachPartition[index] = peersLeft
			peersAlreadyTaken[takenPeer.peer.Hash] = struct{}{}
			takenPeers = append(
				takenPeers, ChosenPeer{Peer: takenPeer.peer, Partition: uint(index)},
			)
			ringFractionsOfTheTakenPeers = append(
				ringFractionsOfTheTakenPeers, takenPeer.ringFractionFromTheWord,
			)
		}
	}

	return takenPeers, ringFractionsOfTheTakenPeers
}

func firstPeerNotYetTaken(
	peersNearestFirst []peerAtRingFractionFromTheWord,
	peersAlreadyTaken map[yacymodel.Hash]struct{},
) (firstPeer peerAtRingFractionFromTheWord, peersLeft []peerAtRingFractionFromTheWord, found bool) {
	for index, firstPeer := range peersNearestFirst {
		if _, taken := peersAlreadyTaken[firstPeer.peer.Hash]; taken {
			continue
		}

		return firstPeer, peersNearestFirst[index+1:], true
	}

	return peerAtRingFractionFromTheWord{}, nil, false
}
