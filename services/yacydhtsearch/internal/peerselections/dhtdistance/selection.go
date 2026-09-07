// Package dhtdistance picks the peers a query goes to by how near they sit to
// the postings of its terms on the DHT ring.
package dhtdistance

import (
	"cmp"
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DHTDistanceObserver interface {
	PeersSelected(ctx context.Context, ringFractions []float64)
}

type Selection struct {
	partitions yacymodel.DHTRingPartitions
	redundancy int
	observer   DHTDistanceObserver
}

func New(
	partitions yacymodel.DHTRingPartitions,
	redundancy int,
	observer DHTDistanceObserver,
) Selection {
	return Selection{partitions: partitions, redundancy: redundancy, observer: observer}
}

func (s Selection) PeersFor(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) []peerdirectory.AskablePeer {
	chosenPeers := make([]peerdirectory.AskablePeer, 0, len(askablePeers))
	ringFractions := make([]float64, 0, len(askablePeers))
	takenPeers := map[yacymodel.Hash]struct{}{}
	for _, term := range query.TermHashes() {
		for partition := range uint(s.partitions) {
			position := yacymodel.DHTRingPositionOfWordInPartition(term, partition, s.partitions)
			for _, peer := range s.nearestPeers(askablePeers, position) {
				if _, seen := takenPeers[peer.Hash]; seen {
					continue
				}
				takenPeers[peer.Hash] = struct{}{}
				chosenPeers = append(chosenPeers, peer)
				ringFractions = append(ringFractions, ringFractionTo(position, peer))
			}
		}
	}
	s.observer.PeersSelected(ctx, ringFractions)

	return chosenPeers
}

func ringFractionTo(
	position yacymodel.DHTRingPosition,
	peer peerdirectory.AskablePeer,
) float64 {
	return position.DistanceTo(yacymodel.DHTRingPositionOf(peer.Hash)).FractionOfDHTRing()
}

func (s Selection) nearestPeers(
	askablePeers []peerdirectory.AskablePeer,
	position yacymodel.DHTRingPosition,
) []peerdirectory.AskablePeer {
	peersByRingDistance := slices.SortedFunc(
		slices.Values(askablePeers),
		func(a, b peerdirectory.AskablePeer) int {
			return cmp.Compare(
				position.DistanceTo(yacymodel.DHTRingPositionOf(a.Hash)),
				position.DistanceTo(yacymodel.DHTRingPositionOf(b.Hash)),
			)
		},
	)

	return peersByRingDistance[:min(s.redundancy, len(peersByRingDistance))]
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
