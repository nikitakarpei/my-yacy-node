package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaulttest"
)

func TestConformance(t *testing.T) {
	vaulttest.RunConformance(t, func(quotaBytes int64) (vault.Engine, error) {
		return memory.OpenEngine(quotaBytes), nil
	})
}

func TestUsedBytesRejectsCancelledContext(t *testing.T) {
	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := v.UsedBytes(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("UsedBytes err = %v, want context.Canceled", err)
	}
}
