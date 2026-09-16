package peerchoice

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	peersThatMayHoldTheWordPerPeerAsked      = 2
	ringsCountedForAPeerTheQueryAlreadyAsked = 1.0
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
	return peersTakenFromEachWordPositionInTurn(
		q.peersNearestToTheWordInEachPartition(queryWord, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord,
	)
}

type peersNearestToWordPosition struct {
	wordPosition yacymodel.DHTRingPosition
	peers        []peerdirectory.AskablePeer
}

func (q peersOneQueryMayAsk) peersNearestToTheWordInEachPartition(
	queryWord yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peersNearestToWordPosition {
	peersAskedByEarlierWords := peersTheQueryAsked(peersOfEarlierWords)
	nearestToEachWordPosition := make([]peersNearestToWordPosition, 0, q.partitions)
	for partition := range uint(q.partitions) {
		wordPosition := yacymodel.DHTRingPositionOfWordInPartition(
			queryWord, partition, q.partitions,
		)
		nearestToEachWordPosition = append(
			nearestToEachWordPosition,
			peersNearestToWordPosition{
				wordPosition: wordPosition,
				peers: q.reliablePeersFirstAmongThoseThatMayHoldTheWord(
					q.peersNearestFirstTo(wordPosition, peersAskedByEarlierWords),
				),
			},
		)
	}

	return nearestToEachWordPosition
}

func peersTheQueryAsked(
	peersOfEarlierWords []peerdirectory.AskablePeer,
) map[yacymodel.Hash]struct{} {
	askedPeers := make(map[yacymodel.Hash]struct{}, len(peersOfEarlierWords))
	for _, peer := range peersOfEarlierWords {
		askedPeers[peer.Hash] = struct{}{}
	}

	return askedPeers
}

func (q peersOneQueryMayAsk) peersNearestFirstTo(
	wordPosition yacymodel.DHTRingPosition,
	peersAskedByEarlierWords map[yacymodel.Hash]struct{},
) []peerdirectory.AskablePeer {
	return slices.SortedFunc(
		slices.Values(q.askablePeers),
		func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
			return cmp.Compare(
				countedRingFractionFrom(wordPosition, firstPeer, peersAskedByEarlierWords),
				countedRingFractionFrom(wordPosition, secondPeer, peersAskedByEarlierWords),
			)
		},
	)
}

func countedRingFractionFrom(
	wordPosition yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
	peersAskedByEarlierWords map[yacymodel.Hash]struct{},
) float64 {
	ringFraction := ringFractionFrom(wordPosition, peer)
	if _, alreadyAsked := peersAskedByEarlierWords[peer.Hash]; alreadyAsked {
		return ringFraction + ringsCountedForAPeerTheQueryAlreadyAsked
	}

	return ringFraction
}

func ringFractionFrom(
	wordPosition yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
) float64 {
	return wordPosition.DistanceTo(
		yacymodel.DHTRingPositionOf(peer.Hash),
	).FractionOfDHTRing()
}

func (q peersOneQueryMayAsk) reliablePeersFirstAmongThoseThatMayHoldTheWord(
	peersNearestFirst []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	amountThatMayHoldTheWord := min(
		len(peersNearestFirst),
		q.amountOfPeersThatMayHoldTheWordAtOneWordPosition(),
	)

	return slices.Concat(
		slices.SortedStableFunc(
			slices.Values(peersNearestFirst[:amountThatMayHoldTheWord]),
			func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
				return cmp.Compare(
					q.reliabilityOfEachPeer[secondPeer.Hash],
					q.reliabilityOfEachPeer[firstPeer.Hash],
				)
			},
		),
		peersNearestFirst[amountThatMayHoldTheWord:],
	)
}

func (q peersOneQueryMayAsk) amountOfPeersThatMayHoldTheWordAtOneWordPosition() int {
	peersAskedAtOneWordPosition := max(1, q.amountOfPeersHoldingOneWord/int(q.partitions))

	return peersAskedAtOneWordPosition * peersThatMayHoldTheWordPerPeerAsked
}

func peersTakenFromEachWordPositionInTurn(
	peersNearestToEachWordPosition []peersNearestToWordPosition,
	peersCeiling int,
) (takenPeers []peerdirectory.AskablePeer, ringFractionsOfTheTakenPeers []float64) {
	peersLeftPerWordPosition := make(
		[][]peerdirectory.AskablePeer, 0, len(peersNearestToEachWordPosition),
	)
	for _, nearest := range peersNearestToEachWordPosition {
		peersLeftPerWordPosition = append(peersLeftPerWordPosition, nearest.peers)
	}
	takenPeers = make([]peerdirectory.AskablePeer, 0, peersCeiling)
	ringFractionsOfTheTakenPeers = make([]float64, 0, peersCeiling)
	peersAlreadyTaken := map[yacymodel.Hash]struct{}{}
	for len(takenPeers) < peersCeiling {
		takenInThisTurn := false
		for index, nearest := range peersNearestToEachWordPosition {
			if len(takenPeers) == peersCeiling {
				break
			}
			peer, peersLeft, found := nearestPeerNotYetTaken(
				peersLeftPerWordPosition[index], peersAlreadyTaken,
			)
			peersLeftPerWordPosition[index] = peersLeft
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

func nearestPeerNotYetTaken(
	peersNearestFirst []peerdirectory.AskablePeer,
	peersAlreadyTaken map[yacymodel.Hash]struct{},
) (nearestPeer peerdirectory.AskablePeer, peersLeft []peerdirectory.AskablePeer, found bool) {
	for index, peer := range peersNearestFirst {
		if _, taken := peersAlreadyTaken[peer.Hash]; taken {
			continue
		}

		return peer, peersNearestFirst[index+1:], true
	}

	return peerdirectory.AskablePeer{}, nil, false
}
