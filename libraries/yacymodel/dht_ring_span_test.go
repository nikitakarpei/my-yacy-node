package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestASpanContainsItsFirstAndLastPositionsAndEveryOneBetween(t *testing.T) {
	span := yacymodel.DHTRingSpan{First: 10, Last: 20}

	for _, position := range []yacymodel.DHTRingPosition{10, 15, 20} {
		if !span.Contains(position) {
			t.Errorf("Contains(%d) = false, want true", position)
		}
	}
}

func TestASpanLeavesOutThePositionsBesideIt(t *testing.T) {
	span := yacymodel.DHTRingSpan{First: 10, Last: 20}

	for _, position := range []yacymodel.DHTRingPosition{0, 9, 21, yacymodel.MaxDHTRingPosition} {
		if span.Contains(position) {
			t.Errorf("Contains(%d) = true, want false", position)
		}
	}
}

func TestASpanOfOnePositionContainsOnlyThatPosition(t *testing.T) {
	span := yacymodel.DHTRingSpan{First: 7, Last: 7}

	if !span.Contains(7) || span.Contains(6) || span.Contains(8) {
		t.Errorf("a span of position 7 must contain only 7")
	}
}
