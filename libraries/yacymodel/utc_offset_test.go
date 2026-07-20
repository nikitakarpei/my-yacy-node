package yacymodel

import (
	"errors"
	"testing"
	"time"
)

func TestNewUTCOffset(t *testing.T) {
	offset, err := NewUTCOffset(150)
	if err != nil || offset.MinutesEast() != 150 {
		t.Fatalf("NewUTCOffset = %d, %v", offset.MinutesEast(), err)
	}
}

func TestNewUTCOffsetRejectsOutOfRange(t *testing.T) {
	for _, m := range []int{-13 * 60, 15 * 60} {
		if _, err := NewUTCOffset(m); !errors.Is(err, ErrBadUTCOffset) {
			t.Fatalf("NewUTCOffset(%d) = %v, want ErrBadUTCOffset", m, err)
		}
	}
}

func TestUTCOffsetOf(t *testing.T) {
	zone := time.FixedZone("test", 2*3600+30*60)
	if got := UTCOffsetOf(time.Date(2026, 7, 21, 0, 0, 0, 0, zone)).MinutesEast(); got != 150 {
		t.Fatalf("UTCOffsetOf = %d, want 150", got)
	}
}
