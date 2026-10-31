package yacymodel_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestWrappedCardinalKeepsTheLowByte(t *testing.T) {
	t.Parallel()

	for cardinal, want := range map[int]uint8{0: 0, 255: 255, 256: 0, 300: 44} {
		if got := yacymodel.WrappedCardinalFrom(cardinal); got != want {
			t.Errorf("wrapped cardinal from %d = %d, want %d", cardinal, got, want)
		}
	}
	if got := yacymodel.WrappedCardinalFrom(uint32(math.MaxUint32)); got != math.MaxUint8 {
		t.Errorf(
			"wrapped cardinal from %d = %d, want %d",
			uint32(math.MaxUint32),
			got,
			math.MaxUint8,
		)
	}
}

func TestSaturatedCardinalStopsAtItsCeiling(t *testing.T) {
	t.Parallel()

	for cardinal, want := range map[int]uint16{
		0:       0,
		65535:   math.MaxUint16,
		65536:   math.MaxUint16,
		1 << 20: math.MaxUint16,
	} {
		if got := yacymodel.SaturatedCardinalFrom[uint16](cardinal); got != want {
			t.Errorf("saturated two-byte cardinal from %d = %d, want %d", cardinal, got, want)
		}
	}
	for cardinal, want := range map[int]uint32{
		0:       0,
		1 << 32: math.MaxUint32,
		1 << 40: math.MaxUint32,
	} {
		if got := yacymodel.SaturatedCardinalFrom[uint32](cardinal); got != want {
			t.Errorf("saturated four-byte cardinal from %d = %d, want %d", cardinal, got, want)
		}
	}
}
