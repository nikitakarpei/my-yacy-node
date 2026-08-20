package nodepeerhash_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodepeerhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

const (
	configuredHashText = "0123456789AB"
	differentHashText  = "BA9876543210"
)

func TestSettleAdoptsTheConfiguredHashAndKeepsIt(t *testing.T) {
	storedData := memory.OpenEngine(1 << 20)

	adopted := settle(t, storedData, yacymodel.Some(mustParseHash(t, configuredHashText)))
	if adopted != mustParseHash(t, configuredHashText) {
		t.Fatalf("Settle = %s, want the configured %s", adopted, configuredHashText)
	}

	kept := settle(t, storedData, yacymodel.None[yacymodel.Hash]())
	if kept != adopted {
		t.Fatalf("Settle after restart = %s, want the adopted %s", kept, adopted)
	}
}

func TestSettleGeneratesAHashWhenNoneIsConfigured(t *testing.T) {
	storedData := memory.OpenEngine(1 << 20)

	generated := settle(t, storedData, yacymodel.None[yacymodel.Hash]())
	if generated == (yacymodel.Hash{}) {
		t.Fatal("Settle returned the zero hash, want a generated one")
	}

	kept := settle(t, storedData, yacymodel.None[yacymodel.Hash]())
	if kept != generated {
		t.Fatalf("Settle after restart = %s, want the generated %s", kept, generated)
	}
}

func TestSettleRefusesAConfiguredHashTheDataDoesNotBelongTo(t *testing.T) {
	storedData := memory.OpenEngine(1 << 20)

	stored := settle(t, storedData, yacymodel.Some(mustParseHash(t, configuredHashText)))

	_, err := selfPeerHashIn(t, storedData).Settle(
		t.Context(),
		yacymodel.Some(mustParseHash(t, differentHashText)),
	)
	if err == nil {
		t.Fatalf("Settle with %s returned nil, want the refusal of %s",
			differentHashText, stored)
	}
}

func settle(
	t *testing.T,
	storedData vault.Engine,
	configuredHash yacymodel.Optional[yacymodel.Hash],
) yacymodel.Hash {
	t.Helper()

	settled, err := selfPeerHashIn(t, storedData).Settle(t.Context(), configuredHash)
	if err != nil {
		t.Fatalf("settle peer hash: %v", err)
	}

	return settled
}

func selfPeerHashIn(t *testing.T, storedData vault.Engine) *nodepeerhash.PeerHash {
	t.Helper()

	storage, err := vault.New(storedData, nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}

	selfPeerHash, err := nodepeerhash.Open(storage)
	if err != nil {
		t.Fatalf("open peer hash: %v", err)
	}

	return selfPeerHash
}

func mustParseHash(t *testing.T, text string) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(text)
	if err != nil {
		t.Fatalf("parse hash %q: %v", text, err)
	}

	return hash
}
