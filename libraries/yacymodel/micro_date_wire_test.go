package yacymodel

import (
	"testing"
	"time"
)

func TestMicroDateWireValueRoundTrip(t *testing.T) {
	original := MicroDateFromTime(time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC))

	parsed, err := ParseMicroDateWireValue(original.WireValue())
	if err != nil {
		t.Fatal(err)
	}
	if parsed != original {
		t.Fatalf("round trip = %d, want %d", parsed, original)
	}
}

func TestMicroDateWireValueWraps(t *testing.T) {
	got, err := ParseMicroDateWireValue("262144")
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("ParseMicroDateWireValue(262144) = %d, want 0", got)
	}
}

func TestParseMicroDateWireValueError(t *testing.T) {
	if _, err := ParseMicroDateWireValue("not-a-number"); err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

func TestMicroDateTimeRoundTrip(t *testing.T) {
	day := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	got := MicroDateFromTime(day).Time()
	if !got.Equal(day) {
		t.Fatalf("Time() = %v, want %v", got, day)
	}
}
