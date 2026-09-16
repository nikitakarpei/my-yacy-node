package peerchoice

import (
	"cmp"
	"math/rand/v2"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfPeersDrawnAtRandomPerQueryWord    = 0.25
	ringsCountedForAPeerTheQueryAlreadyAsked = 1.0
)

type peersOneQueryMayAsk struct {
	partitions                    yacymodel.DHTRingPartitions
	askablePeers                  []peerdirectory.AskablePeer
	shareOfDistanceCountedPerPeer map[yacymodel.Hash]float64
	amountOfPeersHoldingOneWord   int
}

func (q peersOneQueryMayAsk) peersForQueryWord(
	queryWord yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) (
	chosenPeers []peerdirectory.AskablePeer,
	ringFractionsOfTheTakenPeers []float64,
) {
	chosenPeers, ringFractionsOfTheTakenPeers = peersTakenFromEachWordPositionInTurn(
		q.peersNearestToEachPositionOfQueryWord(queryWord, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord-amountOfPeersDrawnAtRandomWithin(
			q.amountOfPeersHoldingOneWord,
		),
	)

	return append(chosenPeers, peersDrawnAtRandomFrom(
		q.askablePeers,
		peersTheQueryAsked(chosenPeers, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord-len(chosenPeers),
	)...), ringFractionsOfTheTakenPeers
}

type peersNearestToWordPosition struct {
	wordPosition yacymodel.DHTRingPosition
	peers        []peerdirectory.AskablePeer
}

func (q peersOneQueryMayAsk) peersNearestToEachPositionOfQueryWord(
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
				peers:        q.peersNearestFirstTo(wordPosition, peersAskedByEarlierWords),
			},
		)
	}

	return nearestToEachWordPosition
}

func peersTheQueryAsked(
	peersPerEarlierChoice ...[]peerdirectory.AskablePeer,
) map[yacymodel.Hash]struct{} {
	askedPeers := map[yacymodel.Hash]struct{}{}
	for _, peers := range peersPerEarlierChoice {
		for _, peer := range peers {
			askedPeers[peer.Hash] = struct{}{}
		}
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
				q.countedRingFractionFrom(wordPosition, firstPeer, peersAskedByEarlierWords),
				q.countedRingFractionFrom(wordPosition, secondPeer, peersAskedByEarlierWords),
			)
		},
	)
}

func (q peersOneQueryMayAsk) countedRingFractionFrom(
	wordPosition yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
	peersAskedByEarlierWords map[yacymodel.Hash]struct{},
) float64 {
	countedRingFraction := ringFractionFrom(wordPosition, peer) *
		q.shareOfDistanceCountedPerPeer[peer.Hash]
	if _, alreadyAsked := peersAskedByEarlierWords[peer.Hash]; alreadyAsked {
		return countedRingFraction + ringsCountedForAPeerTheQueryAlreadyAsked
	}

	return countedRingFraction
}

func ringFractionFrom(
	wordPosition yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
) float64 {
	return wordPosition.DistanceTo(
		yacymodel.DHTRingPositionOf(peer.Hash),
	).FractionOfDHTRing()
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

func amountOfPeersDrawnAtRandomWithin(amountOfPeersHoldingOneWord int) int {
	return int(float64(amountOfPeersHoldingOneWord) * shareOfPeersDrawnAtRandomPerQueryWord)
}

func peersDrawnAtRandomFrom(
	askablePeers []peerdirectory.AskablePeer,
	peersAlreadyChosen map[yacymodel.Hash]struct{},
	amountOfPeersDrawn int,
) []peerdirectory.AskablePeer {
	peersLeft := make([]peerdirectory.AskablePeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		if _, alreadyChosen := peersAlreadyChosen[peer.Hash]; !alreadyChosen {
			peersLeft = append(peersLeft, peer)
		}
	}
	amountDrawn := min(max(amountOfPeersDrawn, 0), len(peersLeft))
	drawn := make([]peerdirectory.AskablePeer, 0, amountDrawn)
	//nolint:gosec // G404: which peers a search explores needs no unpredictability.
	for _, index := range rand.Perm(len(peersLeft))[:amountDrawn] {
		drawn = append(drawn, peersLeft[index])
	}

	return drawn
}
