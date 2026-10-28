package nodepeerhash_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodepeerhash"
)

func TestSettleKeepsTheHashOfTheFirstKey(t *testing.T) {
	storedData := memoryvault.OpenEngine(1 << 20)
	keyHash := peerkeytest.Key(0).Identity().Hash()

	for _, start := range []string{"first start", "restart"} {
		if err := selfPeerHashIn(t, storedData).Settle(t.Context(), keyHash); err != nil {
			t.Fatalf("Settle on %s = %v, want nil", start, err)
		}
	}
}

func TestSettleRefusesTheKeyOfAnotherPeer(t *testing.T) {
	storedData := memoryvault.OpenEngine(1 << 20)
	storedHash := peerkeytest.Key(0).Identity().Hash()
	if err := selfPeerHashIn(t, storedData).Settle(t.Context(), storedHash); err != nil {
		t.Fatalf("settle peer hash: %v", err)
	}

	otherHash := peerkeytest.Key(1).Identity().Hash()
	if err := selfPeerHashIn(t, storedData).Settle(t.Context(), otherHash); err == nil {
		t.Fatalf("Settle with %s = nil, want the refusal of %s", otherHash, storedHash)
	}
}

func selfPeerHashIn(t *testing.T, storedData vault.Engine) *nodepeerhash.PeerHash {
	t.Helper()

	storage, err := vault.New(storedData, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}

	selfPeerHash, err := nodepeerhash.Open(storage)
	if err != nil {
		t.Fatalf("open peer hash: %v", err)
	}

	return selfPeerHash
}
