package contextcancellation_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
)

func TestErrIsNilForLiveContext(t *testing.T) {
	if err := contextcancellation.Err(t.Context()); err != nil {
		t.Fatalf("live context should yield nil, got %v", err)
	}
}

func TestErrWrapsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := contextcancellation.Err(ctx); err == nil {
		t.Fatal("cancelled context should yield an error")
	}
}
