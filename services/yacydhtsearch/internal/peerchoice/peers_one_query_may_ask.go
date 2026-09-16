package peerchoice

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type peersOneQueryMayAsk struct {
	partitions                  yacymodel.DHTRingPartitions
	askablePeers                []peerdirectory.AskablePeer
	reliabilityOfEachPeer       map[yacymodel.Hash]float64
	amountOfPeersHoldingOneWord int
}

func (q peersOneQueryMayAsk) peersForQueryWord(
	queryWord yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) (
	chosenPeers []peerdirectory.AskablePeer,
	ringFractionsOfTheTakenPeers []float64,
) {
	return peersTakenFromEachPartitionInTurn(
		q.peersNearestToTheWordInEachPartition(queryWord, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord,
	)
}

type peersNearestToTheWordInPartition struct {
	wordPosition yacymodel.DHTRingPosition
	peers        []peerdirectory.AskablePeer
}

func (q peersOneQueryMayAsk) peersNearestToTheWordInEachPartition(
	queryWord yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peersNearestToTheWordInPartition {
	peersChosenForEarlierWords := peersChosenForEarlierWordsAmong(
		q.askablePeers,
		peersOfEarlierWords,
	)
	peersNotChosenForEarlierWords := peersNotChosenForEarlierWordsAmong(
		q.askablePeers, peersOfEarlierWords,
	)
	nearestToTheWordInEachPartition := make([]peersNearestToTheWordInPartition, 0, q.partitions)
	for partition := range uint(q.partitions) {
		wordPosition := yacymodel.DHTRingPositionOfWordInPartition(
			queryWord, partition, q.partitions,
		)
		nearestToTheWordInEachPartition = append(
			nearestToTheWordInEachPartition,
			peersNearestToTheWordInPartition{
				wordPosition: wordPosition,
				peers: slices.Concat(
					q.reliablePeersFirstAmongThePeersHoldingTheWord(
						peersNearestFirstTo(wordPosition, peersNotChosenForEarlierWords),
					),
					peersNearestFirstTo(wordPosition, peersChosenForEarlierWords),
				),
			},
		)
	}

	return nearestToTheWordInEachPartition
}

func peersChosenForEarlierWordsAmong(
	askablePeers []peerdirectory.AskablePeer,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	hashesOfPeersChosenForEarlierWords := hashesOf(peersOfEarlierWords)

	return slices.DeleteFunc(
		slices.Clone(askablePeers),
		func(peer peerdirectory.AskablePeer) bool {
			_, chosen := hashesOfPeersChosenForEarlierWords[peer.Hash]

			return !chosen
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

func peersNotChosenForEarlierWordsAmong(
	askablePeers []peerdirectory.AskablePeer,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	hashesOfPeersChosenForEarlierWords := hashesOf(peersOfEarlierWords)

	return slices.DeleteFunc(
		slices.Clone(askablePeers),
		func(peer peerdirectory.AskablePeer) bool {
			_, chosen := hashesOfPeersChosenForEarlierWords[peer.Hash]

			return chosen
		},
	)
}

func (q peersOneQueryMayAsk) reliablePeersFirstAmongThePeersHoldingTheWord(
	peersNearestFirst []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	amountHoldingTheWord := min(
		len(peersNearestFirst),
		max(1, q.amountOfPeersHoldingOneWord/int(q.partitions)),
	)

	return slices.Concat(
		slices.SortedStableFunc(
			slices.Values(peersNearestFirst[:amountHoldingTheWord]),
			func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
				return cmp.Compare(
					q.reliabilityOfEachPeer[secondPeer.Hash],
					q.reliabilityOfEachPeer[firstPeer.Hash],
				)
			},
		),
		peersNearestFirst[amountHoldingTheWord:],
	)
}

func peersNearestFirstTo(
	wordPosition yacymodel.DHTRingPosition,
	peers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.SortedFunc(
		slices.Values(peers),
		func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
			return cmp.Compare(
				ringFractionFrom(wordPosition, firstPeer),
				ringFractionFrom(wordPosition, secondPeer),
			)
		},
	)
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
	peersNearestToTheWordInEachPartition []peersNearestToTheWordInPartition,
	peersCeiling int,
) (takenPeers []peerdirectory.AskablePeer, ringFractionsOfTheTakenPeers []float64) {
	peersLeftPerPartition := make(
		[][]peerdirectory.AskablePeer, 0, len(peersNearestToTheWordInEachPartition),
	)
	for _, nearest := range peersNearestToTheWordInEachPartition {
		peersLeftPerPartition = append(peersLeftPerPartition, nearest.peers)
	}
	takenPeers = make([]peerdirectory.AskablePeer, 0, peersCeiling)
	ringFractionsOfTheTakenPeers = make([]float64, 0, peersCeiling)
	peersAlreadyTaken := map[yacymodel.Hash]struct{}{}
	for len(takenPeers) < peersCeiling {
		takenInThisTurn := false
		for index, nearest := range peersNearestToTheWordInEachPartition {
			if len(takenPeers) == peersCeiling {
				break
			}
			peer, peersLeft, found := firstPeerNotYetTaken(
				peersLeftPerPartition[index], peersAlreadyTaken,
			)
			peersLeftPerPartition[index] = peersLeft
			if !found {
				continue
			}
			peersAlreadyTaken[peer.Hash] = struct{}{}
			takenPeers = append(takenPeers, peer)
			ringFractionsOfTheTakenPeers = append(
				ringFractionsOfTheTakenPeers, ringFractionFrom(nearest.wordPosition, peer),
			)
			takenInThisTurn = true
		}
		if !takenInThisTurn {
			break
		}
	}

	return takenPeers, ringFractionsOfTheTakenPeers
}

func firstPeerNotYetTaken(
	peersNearestFirst []peerdirectory.AskablePeer,
	peersAlreadyTaken map[yacymodel.Hash]struct{},
) (firstPeer peerdirectory.AskablePeer, peersLeft []peerdirectory.AskablePeer, found bool) {
	for index, peer := range peersNearestFirst {
		if _, taken := peersAlreadyTaken[peer.Hash]; taken {
			continue
		}

		return peer, peersNearestFirst[index+1:], true
	}

	return peerdirectory.AskablePeer{}, nil, false
}
