package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestDHTRingSectorOfSpansTheWholeRing(t *testing.T) {
	if sector := yacymodel.DHTRingSectorOf(0); sector != 0 {
		t.Errorf("DHTRingSectorOf(0) = %d, want 0", sector)
	}
	if sector := yacymodel.DHTRingSectorOf(
		yacymodel.MaxDHTRingPosition,
	); sector != yacymodel.MaxDHTRingSector {
		t.Errorf(
			"DHTRingSectorOf(MaxDHTRingPosition) = %d, want %d",
			sector, yacymodel.MaxDHTRingSector,
		)
	}
}

func TestSectorBoundsHoldOnlyPositionsOfThatSector(t *testing.T) {
	for _, sector := range []yacymodel.DHTRingSector{0, 1, yacymodel.MaxDHTRingSector / 2, yacymodel.MaxDHTRingSector} {
		lowest := yacymodel.LowestDHTRingPositionIn(sector)
		highest := yacymodel.HighestDHTRingPositionIn(sector)

		if got := yacymodel.DHTRingSectorOf(lowest); got != sector {
			t.Errorf("lowest position of sector %d sits in sector %d", sector, got)
		}
		if got := yacymodel.DHTRingSectorOf(highest); got != sector {
			t.Errorf("highest position of sector %d sits in sector %d", sector, got)
		}
		if lowest > 0 && yacymodel.DHTRingSectorOf(lowest-1) == sector {
			t.Errorf("position before the lowest of sector %d sits in it", sector)
		}
		if highest < yacymodel.MaxDHTRingPosition &&
			yacymodel.DHTRingSectorOf(highest+1) == sector {
			t.Errorf("position after the highest of sector %d sits in it", sector)
		}
	}
}

func TestSectorBoundsSpanTheWholeRing(t *testing.T) {
	if lowest := yacymodel.LowestDHTRingPositionIn(0); lowest != 0 {
		t.Errorf("lowest position of the first sector = %d, want 0", lowest)
	}
	if highest := yacymodel.HighestDHTRingPositionIn(
		yacymodel.MaxDHTRingSector,
	); highest != yacymodel.MaxDHTRingPosition {
		t.Errorf(
			"highest position of the last sector = %d, want %d",
			highest, yacymodel.MaxDHTRingPosition,
		)
	}
}
