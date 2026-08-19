package peerhash_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

const (
	statedHashText    = "0123456789AB"
	differentHashText = "BA9876543210"
)

func TestSettleAdoptsTheStatedHashAndKeepsIt(t *testing.T) {
	dataDirectory := memory.OpenEngine(1 << 20)

	adopted := settle(t, dataDirectory, yacymodel.Some(mustParseHash(t, statedHashText)))
	if adopted != mustParseHash(t, statedHashText) {
		t.Fatalf("Settle = %s, want the stated %s", adopted, statedHashText)
	}

	kept := settle(t, dataDirectory, yacymodel.None[yacymodel.Hash]())
	if kept != adopted {
		t.Fatalf("Settle after restart = %s, want the adopted %s", kept, adopted)
	}
}

func TestSettleGeneratesAHashWhenNoneIsStated(t *testing.T) {
	dataDirectory := memory.OpenEngine(1 << 20)

	generated := settle(t, dataDirectory, yacymodel.None[yacymodel.Hash]())
	if generated == (yacymodel.Hash{}) {
		t.Fatal("Settle returned the zero hash, want a generated one")
	}

	kept := settle(t, dataDirectory, yacymodel.None[yacymodel.Hash]())
	if kept != generated {
		t.Fatalf("Settle after restart = %s, want the generated %s", kept, generated)
	}
}

func TestSettleRefusesAStatedHashTheDataDoesNotBelongTo(t *testing.T) {
	dataDirectory := memory.OpenEngine(1 << 20)

	stored := settle(t, dataDirectory, yacymodel.Some(mustParseHash(t, statedHashText)))

	_, err := peerhash.Settle(
		t.Context(),
		restarted(t, dataDirectory),
		yacymodel.Some(mustParseHash(t, differentHashText)),
	)
	if err == nil {
		t.Fatalf("Settle with %s returned nil, want the refusal of %s",
			differentHashText, stored)
	}
}

func settle(
	t *testing.T,
	dataDirectory vault.Engine,
	statedHash yacymodel.Optional[yacymodel.Hash],
) yacymodel.Hash {
	t.Helper()

	settled, err := peerhash.Settle(t.Context(), restarted(t, dataDirectory), statedHash)
	if err != nil {
		t.Fatalf("settle peer hash: %v", err)
	}

	return settled
}

func restarted(t *testing.T, dataDirectory vault.Engine) *vault.Vault {
	t.Helper()

	storage, err := vault.New(dataDirectory, nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}

	return storage
}

func mustParseHash(t *testing.T, text string) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(text)
	if err != nil {
		t.Fatalf("parse hash %q: %v", text, err)
	}

	return hash
}
