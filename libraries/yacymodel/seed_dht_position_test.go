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
