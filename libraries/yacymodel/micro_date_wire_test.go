package yacymodel

import (
	"testing"
	"time"
)

func TestMicroDateWireDaysRoundTrip(t *testing.T) {
	original := MicroDateFromTime(time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC))

	if parsed := MicroDateFromWireDays(original.WireDays()); parsed != original {
		t.Fatalf("round trip = %d, want %d", parsed, original)
	}
}

func TestMicroDateFromWireDaysWraps(t *testing.T) {
	if got := MicroDateFromWireDays(microDateWireModulus); got != 0 {
		t.Fatalf("MicroDateFromWireDays(%d) = %d, want 0", microDateWireModulus, got)
	}
}

func TestMicroDateWireDaysWrapsNegative(t *testing.T) {
	if got := MicroDate(-1).WireDays(); got != microDateWireModulus-1 {
		t.Fatalf("WireDays() = %d, want %d", got, microDateWireModulus-1)
	}
}

func TestMicroDateTimeRoundTrip(t *testing.T) {
	day := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	got := MicroDateFromTime(day).Time()
	if !got.Equal(day) {
		t.Fatalf("Time() = %v, want %v", got, day)
	}
}
