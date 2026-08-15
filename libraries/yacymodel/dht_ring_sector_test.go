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
