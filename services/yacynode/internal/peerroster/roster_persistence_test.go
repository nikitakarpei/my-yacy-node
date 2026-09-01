package peerroster_test

import (
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault"
)

func TestRosterRestoresPeerNetworkAddressWithoutRestoringReachability(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "peer-roster.db")
	peerSeed := seniorSeed(t, "peer", "203.0.113.10", 8090)

	storedVault := openBoltRosterVault(t, databasePath)
	storedRoster := openRoster(
		t,
		rosterFixture{storage: storedVault, reservoirCap: 8, reachableCap: 4},
	)
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
	restoredRoster := openRoster(
		t,
		rosterFixture{storage: restoredVault, reservoirCap: 8, reachableCap: 4},
	)
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

	storage, err := boltvault.Open(databasePath, 0, boltvault.WriteBatch{}, nil)
	if err != nil {
		t.Fatalf("boltvault.Open: %v", err)
	}

	return storage
}
