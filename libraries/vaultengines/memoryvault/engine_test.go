package memoryvault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
)

func TestConformance(t *testing.T) {
	vaultenginetest.RunConformance(t, func(quotaBytes int64) (vault.Engine, error) {
		return memoryvault.OpenEngine(quotaBytes), nil
	})
}

func TestUsedBytesRejectsCancelledContext(t *testing.T) {
	v, err := memoryvault.Open(0, nil)
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
