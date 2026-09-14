package dhtdistance_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerselections/dhtdistance"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	partitionExponent = 4
	ringPartitions    = 1 << partitionExponent
	peersCeiling      = 24
)

type recordedFractions struct{ fractions []float64 }

func (r *recordedFractions) PeersSelected(_ context.Context, fractions []float64) {
	r.fractions = fractions
}

func partitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	count, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions from exponent: %v", err)
	}

	return count
}

func askablePeers(t *testing.T, count int) []peerdirectory.AskablePeer {
	t.Helper()

	peers := make([]peerdirectory.AskablePeer, 0, count)
	for i := range count {
		peers = append(peers, peerdirectory.AskablePeer{
			Hash:    yacymodel.WordHash(string(rune('a' + i))),
			Address: "http://10.0.0.1:8090",
		})
	}

	return peers
}

func TestEveryChosenPeerIsNamedOnce(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), &recordedFractions{})

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		askablePeers(t, 20),
		peersCeiling,
	)

	seen := map[yacymodel.Hash]struct{}{}
	for _, peer := range chosen {
		if _, twice := seen[peer.Hash]; twice {
			t.Fatalf("PeersForWord named %s twice", peer.Hash)
		}
		seen[peer.Hash] = struct{}{}
	}
}

func TestNoMorePeersAreChosenThanTheCeilingAllows(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), &recordedFractions{})

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		askablePeers(t, 200),
		peersCeiling,
	)

	if len(chosen) != peersCeiling {
		t.Fatalf("PeersForWord chose %d peers, want %d", len(chosen), peersCeiling)
	}
}

func TestAPartitionOfTheRingKeepsItsPeersWhenTheCeilingIsBelowThePartitions(t *testing.T) {
	t.Parallel()

	const peerOfEachPartition = ringPartitions

	observer := &recordedFractions{}
	selection := dhtdistance.New(partitions(t), observer)

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		askablePeers(t, 200),
		peerOfEachPartition,
	)

	if len(chosen) != peerOfEachPartition {
		t.Fatalf("PeersForWord chose %d peers, want %d", len(chosen), peerOfEachPartition)
	}
	nearestOfEachPartition := 0
	for _, fraction := range observer.fractions {
		if fraction < 1.0/ringPartitions {
			nearestOfEachPartition++
		}
	}
	if nearestOfEachPartition != peerOfEachPartition {
		t.Fatalf(
			"%d of %d chosen peers sit inside their own partition, want all of them",
			nearestOfEachPartition,
			peerOfEachPartition,
		)
	}
}

func TestEveryAskablePeerIsChosenWhenTheyAreFewerThanTheCeiling(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), &recordedFractions{})
	askable := askablePeers(t, 2)

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		askable,
		peersCeiling,
	)

	if len(chosen) != len(askable) {
		t.Fatalf("PeersForWord chose %d of %d askable peers, want all", len(chosen), len(askable))
	}
}

func TestNoPeerIsChosenFromAnEmptyAskableSet(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), &recordedFractions{})

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		nil,
		peersCeiling,
	)

	if len(chosen) != 0 {
		t.Fatalf("PeersForWord = %v, want none", chosen)
	}
}

func TestOneRingFractionIsReportedForEachChosenPeer(t *testing.T) {
	t.Parallel()

	observer := &recordedFractions{}
	selection := dhtdistance.New(partitions(t), dhtdistance.DHTDistanceObservers{observer})

	chosen := selection.PeersForWord(
		t.Context(),
		yacymodel.WordHash("berlin"),
		askablePeers(t, 20),
		peersCeiling,
	)

	if len(observer.fractions) != len(chosen) {
		t.Fatalf(
			"PeersSelected reported %d fractions for %d peers",
			len(observer.fractions),
			len(chosen),
		)
	}
	for _, fraction := range observer.fractions {
		if fraction < 0 || fraction > 1 {
			t.Fatalf("ring fraction %v is outside the ring", fraction)
		}
	}
}
