package peerchoice

import (
	"cmp"
	"context"
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
	observer                      PeerChoiceObserver
	askablePeers                  []peerdirectory.AskablePeer
	shareOfDistanceCountedPerPeer map[peerdirectory.AskablePeer]float64
	amountOfPeersHoldingOneWord   int
}

func (q peersOneQueryMayAsk) peersForQueryWord(
	ctx context.Context,
	queryWord yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	takenPeers := peersTakenFromEachWordPositionInTurn(
		q.peersNearestToEachPositionOfQueryWord(queryWord, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord-amountOfPeersDrawnAtRandomWithin(
			q.amountOfPeersHoldingOneWord,
		),
	)
	q.observer.PeersTakenFromTheRing(ctx, ringFractionsOf(takenPeers))
	chosenPeers := peersOf(takenPeers)

	return append(chosenPeers, peersDrawnAtRandomFrom(
		q.askablePeers,
		peersTheQueryAsked(chosenPeers, peersOfEarlierWords),
		q.amountOfPeersHoldingOneWord-len(chosenPeers),
	)...)
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
) map[peerdirectory.AskablePeer]struct{} {
	askedPeers := map[peerdirectory.AskablePeer]struct{}{}
	for _, peers := range peersPerEarlierChoice {
		for _, peer := range peers {
			askedPeers[peer] = struct{}{}
		}
	}

	return askedPeers
}

func (q peersOneQueryMayAsk) peersNearestFirstTo(
	wordPosition yacymodel.DHTRingPosition,
	peersAskedByEarlierWords map[peerdirectory.AskablePeer]struct{},
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
	peersAskedByEarlierWords map[peerdirectory.AskablePeer]struct{},
) float64 {
	countedRingFraction := ringFractionFrom(wordPosition, peer) *
		q.shareOfDistanceCountedPerPeer[peer]
	if _, alreadyAsked := peersAskedByEarlierWords[peer]; alreadyAsked {
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

type peerTakenFromWordPosition struct {
	peer         peerdirectory.AskablePeer
	wordPosition yacymodel.DHTRingPosition
}

func peersTakenFromEachWordPositionInTurn(
	peersNearestToEachWordPosition []peersNearestToWordPosition,
	peersCeiling int,
) []peerTakenFromWordPosition {
	peersLeftPerWordPosition := make(
		[][]peerdirectory.AskablePeer, 0, len(peersNearestToEachWordPosition),
	)
	for _, nearest := range peersNearestToEachWordPosition {
		peersLeftPerWordPosition = append(peersLeftPerWordPosition, nearest.peers)
	}
	takenPeers := make([]peerTakenFromWordPosition, 0, peersCeiling)
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
			takenPeers = append(takenPeers, peerTakenFromWordPosition{
				peer:         peer,
				wordPosition: nearest.wordPosition,
			})
			takenInThisTurn = true
		}
		if !takenInThisTurn {
			break
		}
	}

	return takenPeers
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

func ringFractionsOf(takenPeers []peerTakenFromWordPosition) []float64 {
	fractions := make([]float64, 0, len(takenPeers))
	for _, taken := range takenPeers {
		fractions = append(fractions, ringFractionFrom(taken.wordPosition, taken.peer))
	}

	return fractions
}

func peersOf(takenPeers []peerTakenFromWordPosition) []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(takenPeers))
	for _, taken := range takenPeers {
		peers = append(peers, taken.peer)
	}

	return peers
}

func peersDrawnAtRandomFrom(
	askablePeers []peerdirectory.AskablePeer,
	peersAlreadyChosen map[peerdirectory.AskablePeer]struct{},
	amountOfPeersDrawn int,
) []peerdirectory.AskablePeer {
	peersLeft := make([]peerdirectory.AskablePeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		if _, alreadyChosen := peersAlreadyChosen[peer]; !alreadyChosen {
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
