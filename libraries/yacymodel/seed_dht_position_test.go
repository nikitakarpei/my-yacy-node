package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func seedAt(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{Hash: hash}
}

func TestDistanceToIsZeroAtTheSamePosition(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.DHTRingPositionOf(near)

	if got := position.DistanceTo(yacymodel.DHTRingPositionOf(near)); got != 0 {
		t.Fatalf("distance = %d, want 0 for the hash the position came from", got)
	}
	if position.DistanceTo(yacymodel.DHTRingPositionOf(far)) == 0 {
		t.Fatalf("distance = 0, want a nonzero distance for a different hash")
	}
}

func TestFractionOfDHTRingIsZeroAtTheSamePosition(t *testing.T) {
	near := yacymodel.WordHash("near")
	far := yacymodel.WordHash("far")

	position := yacymodel.DHTRingPositionOf(near)

	got := position.DistanceTo(yacymodel.DHTRingPositionOf(near)).FractionOfDHTRing()
	if got != 0 {
		t.Fatalf("fraction = %v, want 0 for the hash the position came from", got)
	}

	fraction := position.DistanceTo(yacymodel.DHTRingPositionOf(far)).FractionOfDHTRing()
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

	occupied := yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOf(peer))
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
