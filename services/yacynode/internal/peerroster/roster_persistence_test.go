package peerroster_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

func TestRosterRestoresPeerNetworkAddressWithoutRestoringReachability(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "peer-roster.db")
	peerSeed := seniorSeed(t, "peer", "203.0.113.10", 8090)

	storedVault := openBoltRosterVault(t, databasePath)
	storedRoster := openRosterOver(t, storedVault)
	storedRoster.Discover(t.Context(), peerSeed)
	if err := storedVault.Close(); err != nil {
		t.Fatalf("Close stored roster: %v", err)
	}

	restoredVault := openBoltRosterVault(t, databasePath)
	t.Cleanup(func() {
		if err := restoredVault.Close(); err != nil {
			t.Errorf("Close restored roster: %v", err)
		}
	})
	restoredRoster := openRosterOver(t, restoredVault)
	if got := restoredRoster.ReachablePeers(t.Context()); len(got) != 0 {
		t.Fatalf("reachable peers after restart = %d, want 0", len(got))
	}
	if _, found := hashSet(restoredRoster.UnreachablePeerHashes(t.Context(), 1))[peerSeed.Hash]; !found {
		t.Fatalf("persisted peer missing from unreachable peer hashes")
	}

	restoredAddress, found := restoredRoster.NetworkAddressOf(t.Context(), peerSeed.Hash)
	if !found {
		t.Fatal("persisted peer network address not found")
	}
	wantAddress, _ := peerSeed.NetworkAddress()
	if restoredAddress != wantAddress {
		t.Errorf("network address = %v, want %v", restoredAddress, wantAddress)
	}
}

func openBoltRosterVault(t testing.TB, databasePath string) *vault.Vault {
	t.Helper()

	storage, err := boltvault.Open(databasePath, 0, nil)
	if err != nil {
		t.Fatalf("boltvault.Open: %v", err)
	}

	return storage
}

func openRosterOver(t testing.TB, storage *vault.Vault) peerroster.Roster {
	t.Helper()

	roster, err := peerroster.Open(
		storage,
		func() time.Time { return time.Unix(2_000, 0).UTC() },
		8,
		4,
		defaultAnnounceInterval,
		selfHash(),
		peerroster.DiscardObserver,
	)
	if err != nil {
		t.Fatalf("peerroster.Open: %v", err)
	}

	return roster
}
