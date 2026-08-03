package yacymodel

import "testing"

func TestDHTRingSectorOfSpansTheWholeRing(t *testing.T) {
	if sector := DHTRingSectorOf(0); sector != 0 {
		t.Errorf("DHTRingSectorOf(0) = %d, want 0", sector)
	}
	if sector := DHTRingSectorOf(MaxDHTPosition); sector != MaxDHTRingSector {
		t.Errorf(
			"DHTRingSectorOf(MaxDHTPosition) = %d, want %d",
			sector, MaxDHTRingSector,
		)
	}
}

func TestRingFractionOfDistance(t *testing.T) {
	if fraction := ringFractionOfDistance(0); fraction != 0 {
		t.Errorf("ringFractionOfDistance(0) = %v, want 0", fraction)
	}

	if fraction := ringFractionOfDistance(MaxDHTPosition / 2); fraction < 0.49 || fraction > 0.51 {
		t.Errorf("ringFractionOfDistance(MaxDHTPosition/2) = %v, want about 0.5", fraction)
	}
}
