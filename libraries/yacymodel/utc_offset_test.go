package yacymodel_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestUTCOffsetCarriesTheMinutesEastItWasMadeWith(t *testing.T) {
	t.Parallel()

	offset, err := yacymodel.NewUTCOffset(150)
	if err != nil || offset.MinutesEast() != 150 {
		t.Fatalf("NewUTCOffset = %d, %v", offset.MinutesEast(), err)
	}
}

func TestNewUTCOffsetRejectsAnOffsetOutsideTheZoneRange(t *testing.T) {
	t.Parallel()

	for _, minutes := range []int{-13 * 60, 15 * 60} {
		if _, err := yacymodel.NewUTCOffset(minutes); !errors.Is(err, yacymodel.ErrBadUTCOffset) {
			t.Fatalf("NewUTCOffset(%d) = %v, want ErrBadUTCOffset", minutes, err)
		}
	}
}

func TestUTCOffsetOfATimeReadsItsZone(t *testing.T) {
	t.Parallel()

	zone := time.FixedZone("test", 2*3600+30*60)
	zoned := time.Date(2026, 7, 21, 0, 0, 0, 0, zone)

	if got := yacymodel.UTCOffsetOf(zoned).MinutesEast(); got != 150 {
		t.Fatalf("UTCOffsetOf = %d, want 150", got)
	}
}
