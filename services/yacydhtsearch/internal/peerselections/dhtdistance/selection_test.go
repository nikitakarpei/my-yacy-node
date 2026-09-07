package dhtdistance_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerselections/dhtdistance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const partitionExponent = 4

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

	selection := dhtdistance.New(partitions(t), 3, &recordedFractions{})

	chosen := selection.PeersFor(
		t.Context(),
		searchquery.QueryFrom("berlin weather forecast"),
		askablePeers(t, 20),
	)

	seen := map[yacymodel.Hash]struct{}{}
	for _, peer := range chosen {
		if _, twice := seen[peer.Hash]; twice {
			t.Fatalf("PeersFor named %s twice", peer.Hash)
		}
		seen[peer.Hash] = struct{}{}
	}
}

func TestEveryAskablePeerIsChosenWhenTheyAreFewerThanTheRedundancy(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), 3, &recordedFractions{})
	askable := askablePeers(t, 2)

	chosen := selection.PeersFor(t.Context(), searchquery.QueryFrom("berlin"), askable)

	if len(chosen) != len(askable) {
		t.Fatalf("PeersFor chose %d of %d askable peers, want all", len(chosen), len(askable))
	}
}

func TestNoPeerIsChosenFromAnEmptyAskableSet(t *testing.T) {
	t.Parallel()

	selection := dhtdistance.New(partitions(t), 3, &recordedFractions{})

	if chosen := selection.PeersFor(
		t.Context(),
		searchquery.QueryFrom("berlin"),
		nil,
	); len(
		chosen,
	) != 0 {
		t.Fatalf("PeersFor = %v, want none", chosen)
	}
}

func TestOneRingFractionIsReportedForEachChosenPeer(t *testing.T) {
	t.Parallel()

	observer := &recordedFractions{}
	selection := dhtdistance.New(partitions(t), 3, dhtdistance.DHTDistanceObservers{observer})

	chosen := selection.PeersFor(t.Context(), searchquery.QueryFrom("berlin"), askablePeers(t, 20))

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
