// Package dhtdistance picks the peers one word of a query goes to by how near
// they sit to the postings of that word on the DHT ring. It takes the peers
// from every partition of the ring in turn, so that a ceiling below what the
// partitions hold still leaves every partition covered.
package dhtdistance

import (
	"cmp"
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DHTDistanceObserver interface {
	PeersSelected(ctx context.Context, ringFractions []float64)
}

type Selection struct {
	partitions yacymodel.DHTRingPartitions
	observer   DHTDistanceObserver
}

func New(
	partitions yacymodel.DHTRingPartitions,
	observer DHTDistanceObserver,
) Selection {
	return Selection{partitions: partitions, observer: observer}
}

func (s Selection) PeersForWord(
	ctx context.Context,
	word yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
	peersCeiling int,
) []peerdirectory.AskablePeer {
	peersNearestToEachPosting := s.peersNearestToEachPostingOf(word, askablePeers)
	nearestPeers := peersTakenFromEachPostingInTurn(peersNearestToEachPosting, peersCeiling)
	s.observer.PeersSelected(ctx, ringFractionsOf(nearestPeers))

	return peersOf(nearestPeers)
}

type peersNearestToPosting struct {
	posting yacymodel.DHTRingPosition
	peers   []peerdirectory.AskablePeer
}

func (s Selection) peersNearestToEachPostingOf(
	word yacymodel.Hash,
	askablePeers []peerdirectory.AskablePeer,
) []peersNearestToPosting {
	nearestToEachPosting := make([]peersNearestToPosting, 0, s.partitions)
	for partition := range uint(s.partitions) {
		posting := yacymodel.DHTRingPositionOfWordInPartition(word, partition, s.partitions)
		nearestToEachPosting = append(nearestToEachPosting, peersNearestToPosting{
			posting: posting,
			peers:   peersByRingDistanceTo(posting, askablePeers),
		})
	}

	return nearestToEachPosting
}

func peersByRingDistanceTo(
	posting yacymodel.DHTRingPosition,
	askablePeers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	return slices.SortedFunc(
		slices.Values(askablePeers),
		func(firstPeer, secondPeer peerdirectory.AskablePeer) int {
			return cmp.Compare(
				posting.DistanceTo(yacymodel.DHTRingPositionOf(firstPeer.Hash)),
				posting.DistanceTo(yacymodel.DHTRingPositionOf(secondPeer.Hash)),
			)
		},
	)
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
		fractions = append(fractions, nearest.posting.DistanceTo(
			yacymodel.DHTRingPositionOf(nearest.peer.Hash),
		).FractionOfDHTRing())
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

type DHTDistanceObservers []DHTDistanceObserver

func (observers DHTDistanceObservers) PeersSelected(
	ctx context.Context,
	ringFractions []float64,
) {
	for _, observer := range observers {
		observer.PeersSelected(ctx, ringFractions)
	}
}
