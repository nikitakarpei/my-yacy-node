package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func seedAt(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{Hash: hash}
}

func TestDistanceToPositionIsZeroAtThePosition(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.RingPosition(near)

	if got := yacymodel.DistanceToPosition(near, position); got != 0 {
		t.Fatalf("distance = %d, want 0 for the hash the position came from", got)
	}
	if yacymodel.DistanceToPosition(far, position) == 0 {
		t.Fatalf("distance = 0, want a nonzero distance for a different hash")
	}
}

func TestSeedsClosestToPositionOrdersByRingDistance(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.RingPosition(near)

	closest := yacymodel.SeedsClosestToPosition(
		[]yacymodel.Seed{seedAt(far), seedAt(near)}, position, 2,
	)
	if len(closest) != 2 || closest[0].Hash != near {
		t.Fatalf("closest = %v, want the seed at the position first", closest)
	}
}

func TestSeedsClosestToPositionCapsToWant(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.RingPosition(near)

	closest := yacymodel.SeedsClosestToPosition(
		[]yacymodel.Seed{seedAt(far), seedAt(near)}, position, 1,
	)
	if len(closest) != 1 || closest[0].Hash != near {
		t.Fatalf("closest = %v, want only the nearest seed", closest)
	}
}

func TestSeedsClosestToPositionWithoutWantSelectsNothing(t *testing.T) {
	closest := yacymodel.SeedsClosestToPosition(
		[]yacymodel.Seed{seedAt(yacymodel.WordHash("near"))}, 0, 0,
	)
	if len(closest) != 0 {
		t.Fatalf("closest = %v, want nothing when no seed is wanted", closest)
	}
}

func TestSeedsCloserThanPeerKeepsOnlyTheCloserSeeds(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.RingPosition(near)

	closer := yacymodel.SeedsCloserThanPeer(
		[]yacymodel.Seed{seedAt(far), seedAt(near)}, far, position,
	)
	if len(closer) != 1 || closer[0].Hash != near {
		t.Fatalf("closer = %v, want only the seed closer than the peer", closer)
	}
}

func TestRingFractionToPositionIsZeroAtThePosition(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.RingPosition(near)

	if got := yacymodel.RingFractionToPosition(near, position); got != 0 {
		t.Fatalf("fraction = %v, want 0 for the hash the position came from", got)
	}
	fraction := yacymodel.RingFractionToPosition(far, position)
	if fraction <= 0 || fraction >= 1 {
		t.Fatalf("fraction = %v, want a fraction in (0,1)", fraction)
	}
}

func TestSeedsPerDHTRingSectorCountsEverySector(t *testing.T) {
	peer := yacymodel.WordHash("peer")

	perSector := yacymodel.SeedsPerDHTRingSector([]yacymodel.Seed{seedAt(peer)})

	if len(perSector) != int(yacymodel.MaxDHTRingSector)+1 {
		t.Fatalf("sectors = %d, want %d", len(perSector), yacymodel.MaxDHTRingSector+1)
	}

	occupied := yacymodel.DHTRingSectorOf(yacymodel.RingPosition(peer))
	for sector, seeds := range perSector {
		want := 0
		if yacymodel.DHTRingSector(sector) == occupied {
			want = 1
		}
		if seeds != want {
			t.Errorf("sector %d holds %d seeds, want %d", sector, seeds, want)
		}
	}
}
