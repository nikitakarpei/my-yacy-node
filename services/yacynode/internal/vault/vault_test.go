package vault_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

func TestNilEngineRejected(t *testing.T) {
	if _, err := vault.New(nil, nil); err == nil {
		t.Fatal("New(nil) succeeded, want error")
	}
}

func TestAtCapacityIgnoresUnsetQuota(t *testing.T) {
	v, _ := openWords(t)

	atCapacity, err := v.AtCapacity(context.Background())
	if err != nil {
		t.Fatalf("AtCapacity: %v", err)
	}
	if atCapacity {
		t.Fatal("AtCapacity = true, want false without a quota")
	}
}

func TestClosedVaultRejectsOperations(t *testing.T) {
	ctx := context.Background()
	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	if err := v.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := v.Update(ctx, func(*vault.Txn) error { return nil }); err == nil {
		t.Fatal("Update on closed vault succeeded, want error")
	}
	if err := v.View(ctx, func(*vault.Txn) error { return nil }); err == nil {
		t.Fatal("View on closed vault succeeded, want error")
	}
	if _, err := v.UsedBytes(ctx); err == nil {
		t.Fatal("UsedBytes on closed vault succeeded, want error")
	}
	if _, err := v.AtCapacity(ctx); err == nil {
		t.Fatal("AtCapacity on closed vault succeeded, want error")
	}
	if err := v.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
