package peerchoice

import (
	"cmp"
	"context"
	"math/rand/v2"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfPeersDrawnAtRandomPerQueryWord = 0.25

type choiceForOneQuery struct {
	Choice
	askablePeers                  []peerdirectory.AskablePeer
	shareOfDistanceCountedPerPeer map[peerdirectory.AskablePeer]float64
	peersCeiling                  int
}

func (q choiceForOneQuery) peersForWord(
	ctx context.Context,
	word yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	nearestPeers := peersTakenFromEachPostingInTurn(
		q.peersNearestToEachPostingOf(word, peersOfEarlierWords),
		q.peersCeiling-peersDrawnAtRandomWithin(q.peersCeiling),
	)
	q.observer.PeersSelected(ctx, ringFractionsOf(nearestPeers))
	chosenPeers := peersOf(nearestPeers)

	return append(chosenPeers, peersDrawnAtRandomFrom(
		q.askablePeers,
		slices.Concat(chosenPeers, peersOfEarlierWords),
		q.peersCeiling-len(chosenPeers),
	)...)
}

type peersNearestToPosting struct {
	posting yacymodel.DHTRingPosition
	peers   []peerdirectory.AskablePeer
}

func (q choiceForOneQuery) peersNearestToEachPostingOf(
	word yacymodel.Hash,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) []peersNearestToPosting {
	peersTheQueryHasNotAsked, peersTheQueryHasAsked := peersApartByWhatTheQueryAsked(
		q.askablePeers, peersOfEarlierWords,
	)
	nearestToEachPosting := make([]peersNearestToPosting, 0, q.partitions)
	for partition := range uint(q.partitions) {
		posting := yacymodel.DHTRingPositionOfWordInPartition(word, partition, q.partitions)
		nearestToEachPosting = append(nearestToEachPosting, peersNearestToPosting{
			posting: posting,
			peers: slices.Concat(
				q.peersByCountedRingFractionTo(posting, peersTheQueryHasNotAsked),
				q.peersByCountedRingFractionTo(posting, peersTheQueryHasAsked),
			),
		})
	}

	return nearestToEachPosting
}

func peersApartByWhatTheQueryAsked(
	askablePeers []peerdirectory.AskablePeer,
	peersOfEarlierWords []peerdirectory.AskablePeer,
) (notAsked, asked []peerdirectory.AskablePeer) {
	peersAlreadyAsked := setOf(peersOfEarlierWords)
	for _, peer := range askablePeers {
		if _, alreadyAsked := peersAlreadyAsked[peer]; alreadyAsked {
			asked = append(asked, peer)

			continue
		}
		notAsked = append(notAsked, peer)
	}

	return notAsked, asked
}

func setOf(peers []peerdirectory.AskablePeer) map[peerdirectory.AskablePeer]struct{} {
	set := make(map[peerdirectory.AskablePeer]struct{}, len(peers))
	for _, peer := range peers {
		set[peer] = struct{}{}
	}

	return set
}

func (q choiceForOneQuery) peersByCountedRingFractionTo(
	posting yacymodel.DHTRingPosition,
	peers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.SortedFunc(
		slices.Values(peers),
		func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
			return cmp.Compare(
				ringFractionFrom(posting, firstPeer)*
					q.shareOfDistanceCountedPerPeer[firstPeer],
				ringFractionFrom(posting, secondPeer)*
					q.shareOfDistanceCountedPerPeer[secondPeer],
			)
		},
	)
}

func ringFractionFrom(
	posting yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
) float64 {
	return posting.DistanceTo(yacymodel.DHTRingPositionOf(peer.Hash)).FractionOfDHTRing()
}

type peerNearestToPosting struct {
	peer    peerdirectory.AskablePeer
	posting yacymodel.DHTRingPosition
}

func peersTakenFromEachPostingInTurn(
	peersNearestToEachPosting []peersNearestToPosting,
	peersCeiling int,
) []peerNearestToPosting {
	takenPeers := make([]peerNearestToPosting, 0, peersCeiling)
	peersAlreadyTaken := map[yacymodel.Hash]struct{}{}
	nextPeerPerPosting := make([]int, len(peersNearestToEachPosting))
	for takenInTurn := -1; takenInTurn != 0 && len(takenPeers) < peersCeiling; {
		takenInTurn = 0
		for index, nearest := range peersNearestToEachPosting {
			if len(takenPeers) == peersCeiling {
				break
			}
			peer, nextPeer, found := peerNotYetTaken(
				nearest.peers, nextPeerPerPosting[index], peersAlreadyTaken,
			)
			nextPeerPerPosting[index] = nextPeer
			if !found {
				continue
			}
			peersAlreadyTaken[peer.Hash] = struct{}{}
			takenPeers = append(
				takenPeers,
				peerNearestToPosting{peer: peer, posting: nearest.posting},
			)
			takenInTurn++
		}
	}

	return takenPeers
}

func peerNotYetTaken(
	peers []peerdirectory.AskablePeer,
	nextPeer int,
	peersAlreadyTaken map[yacymodel.Hash]struct{},
) (peerdirectory.AskablePeer, int, bool) {
	for index := nextPeer; index < len(peers); index++ {
		if _, taken := peersAlreadyTaken[peers[index].Hash]; taken {
			continue
		}

		return peers[index], index + 1, true
	}

	return peerdirectory.AskablePeer{}, len(peers), false
}

func ringFractionsOf(nearestPeers []peerNearestToPosting) []float64 {
	fractions := make([]float64, 0, len(nearestPeers))
	for _, nearest := range nearestPeers {
		fractions = append(fractions, ringFractionFrom(nearest.posting, nearest.peer))
	}

	return fractions
}

func peersOf(nearestPeers []peerNearestToPosting) []peerdirectory.AskablePeer {
	peers := make([]peerdirectory.AskablePeer, 0, len(nearestPeers))
	for _, nearest := range nearestPeers {
		peers = append(peers, nearest.peer)
	}

	return peers
}

func peersDrawnAtRandomWithin(peersCeiling int) int {
	return int(float64(peersCeiling) * shareOfPeersDrawnAtRandomPerQueryWord)
}

func peersDrawnAtRandomFrom(
	askablePeers []peerdirectory.AskablePeer,
	peersAlreadyChosen []peerdirectory.AskablePeer,
	amountOfPeers int,
) []peerdirectory.AskablePeer {
	chosen := setOf(peersAlreadyChosen)
	peersLeft := make([]peerdirectory.AskablePeer, 0, len(askablePeers))
	for _, peer := range askablePeers {
		if _, alreadyChosen := chosen[peer]; !alreadyChosen {
			peersLeft = append(peersLeft, peer)
		}
	}
	amountDrawn := min(max(amountOfPeers, 0), len(peersLeft))
	drawn := make([]peerdirectory.AskablePeer, 0, amountDrawn)
	//nolint:gosec // G404: which peers a search explores needs no unpredictability.
	for _, index := range rand.Perm(len(peersLeft))[:amountDrawn] {
		drawn = append(drawn, peersLeft[index])
	}

	return drawn
}
