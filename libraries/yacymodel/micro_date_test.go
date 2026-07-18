package yacymodel

import (
	"testing"
	"time"
)

func TestMicroDateTimeRoundTrip(t *testing.T) {
	day := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	got := MicroDateFromTime(day).Time()
	if !got.Equal(day) {
		t.Fatalf("Time() = %v, want %v", got, day)
	}
}
