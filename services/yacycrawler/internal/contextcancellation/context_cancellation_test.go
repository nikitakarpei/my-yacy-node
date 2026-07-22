package contextcancellation

import (
	"context"
	"testing"
)

func TestErrIsNilForLiveContext(t *testing.T) {
	if err := Err(t.Context()); err != nil {
		t.Fatalf("live context should yield nil, got %v", err)
	}
}

func TestErrWrapsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Err(ctx); err == nil {
		t.Fatal("cancelled context should yield an error")
	}
}
