package wallclock

import (
	"context"
	"testing"
	"time"
)

func TestNowReportsCurrentTime(t *testing.T) {
	before := time.Now()
	now := Clock{}.Now()
	if now.Before(before) {
		t.Fatalf("Now %v predates the call %v", now, before)
	}
}

func TestSleepReturnsAfterDuration(t *testing.T) {
	start := time.Now()
	if err := (Clock{}).Sleep(t.Context(), time.Millisecond); err != nil {
		t.Fatalf("sleep: %v", err)
	}
	if time.Since(start) < time.Millisecond {
		t.Fatal("sleep returned before the duration elapsed")
	}
}

func TestSleepNonPositiveDurationReturnsNil(t *testing.T) {
	if err := (Clock{}).Sleep(t.Context(), 0); err != nil {
		t.Fatalf("zero duration should return nil on a live context: %v", err)
	}
}

func TestSleepWrapsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (Clock{}).Sleep(ctx, 0); err == nil {
		t.Fatal("cancelled context should surface an error")
	}
	if err := (Clock{}).Sleep(ctx, time.Hour); err == nil {
		t.Fatal("cancelled context should interrupt a pending sleep")
	}
}
